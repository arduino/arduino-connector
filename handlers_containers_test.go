//
//  This file is part of arduino-connector
//
//  Copyright (C) 2017-2018  Arduino AG (http://www.arduino.cc/)
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//

package main

import (
	"os/exec"
	"strings"
    "sync"
    "testing"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"fmt"
	"crypto/tls"

)

func TestConnectorProcessIsRunning(t *testing.T) {
	vagrantCmd:="systemctl status ArduinoConnector | grep running"
	vagrantSSHCmd := fmt.Sprintf(`cd test && vagrant ssh -c "%s"`,vagrantCmd)
	cmd := exec.Command("bash", "-c", vagrantSSHCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Error(err)
	}
	outputMessage:=string(out)
	if !strings.Contains(outputMessage, "active (running)") {
		t.Error(outputMessage)
	}
}

func TestConnectorDockerIsRunning(t *testing.T) {
	vagrantCmd:="sudo docker version"
	vagrantSSHCmd := fmt.Sprintf(`cd test && vagrant ssh -c "%s"`,vagrantCmd)
	cmd := exec.Command("bash", "-c", vagrantSSHCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Error(err)
	}
	outputMessage:=string(out)
	if !strings.Contains(outputMessage, "Version:") {
		t.Error(outputMessage)
	}
}


func TestContainerStatus(t *testing.T){

	const TOPIC = "mytopic/test"
	url:="a19g5nbe27wn47.iot.us-east-1.amazonaws.com"
	port := 8883
	path := "/mqtt"
	brokerURL := fmt.Sprintf("tcps://%s:%d%s", url, port, path)
	cert:="test/cert.pem"
	key:="test/privateKey.pem"
	id:="testThingVagrant"
	cer, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		t.Error(err, "read certificate")
	}
	opts := mqtt.NewClientOptions().AddBroker(brokerURL)
	opts.SetClientID(id)
	opts.SetTLSConfig(&tls.Config{
		Certificates: []tls.Certificate{cer},
		ServerName:   url,
	})


	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
			t.Fatal(token.Error())
	}

	var wg sync.WaitGroup
	wg.Add(1)

	if token := client.Subscribe(TOPIC, 0, func(client mqtt.Client, msg mqtt.Message) {
			if string(msg.Payload()) != "mymessagee" {
				    wg.Done()
					t.Fatalf("want mymessagee, got %s", msg.Payload())
			}
			wg.Done()
	}); token.Wait() && token.Error() != nil {
			t.Fatal(token.Error())
	}

	if token := client.Publish(TOPIC, 0, false, "mymessagee"); token.Wait() && token.Error() != nil {
			t.Fatal(token.Error())
	}
	wg.Wait()

}
