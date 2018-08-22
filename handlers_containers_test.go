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
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"testing"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// testing helpers
// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}

// MqttTestClient is an ad-hoc mqtt client struct for test
type MqttTestClient struct {
	client mqtt.Client
}

func NewMqttTestClient() *MqttTestClient {
	cert := "test/cert.pem"
	key := "test/privateKey.pem"
	id := "testThingVagrant"
	port := 8883
	path := "/mqtt"
	file, err := ioutil.ReadFile("test/cert_arn.sh")
	if err != nil {
        panic(err)
	}
	url:="endpoint.iot.com"
    for _,line := range strings.Split(string(file),"\n"){
		if strings.Contains(line,"IOT_ENDPOINT"){
			url=strings.Split(line,"=")[1]
		}
	}
	brokerURL := fmt.Sprintf("tcps://%s:%d%s", url, port, path)
	cer, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		panic(err)
	}
	opts := mqtt.NewClientOptions().AddBroker(brokerURL)
	opts.SetClientID(id)
	opts.SetTLSConfig(&tls.Config{
		Certificates: []tls.Certificate{cer},
		ServerName:   url,
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	return &MqttTestClient{client}
}

func (tmc *MqttTestClient) Close() {
	tmc.client.Disconnect(100)
}

func (tmc *MqttTestClient) MqttSendAndReceiveSync(t *testing.T, topic, request, goldResponse string) {

	var wg sync.WaitGroup
	wg.Add(1)

	if token := tmc.client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		if string(msg.Payload()) != goldResponse {
			wg.Done()
			t.Fatalf("want %s, got %s", goldResponse, msg.Payload())
		}
		wg.Done()
	}); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}

	if token := tmc.client.Publish(topic, 0, false, request); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}
	wg.Wait()

}

// tests
func TestConnectorProcessIsRunning(t *testing.T) {
	vagrantCmd := "systemctl status ArduinoConnector | grep running"
	vagrantSSHCmd := fmt.Sprintf(`cd test && vagrant ssh -c "%s"`, vagrantCmd)
	cmd := exec.Command("bash", "-c", vagrantSSHCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Error(err)
	}
	outputMessage := string(out)
	if !strings.Contains(outputMessage, "active (running)") {
		t.Error(outputMessage)
	}
}

func TestConnectorDockerIsRunning(t *testing.T) {
	vagrantCmd := "sudo docker version"
	vagrantSSHCmd := fmt.Sprintf(`cd test && vagrant ssh -c "%s"`, vagrantCmd)
	cmd := exec.Command("bash", "-c", vagrantSSHCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Error(err)
	}
	outputMessage := string(out)
	if !strings.Contains(outputMessage, "Version:") {
		t.Error(outputMessage)
	}
}

func TestContainerStatus(t *testing.T) {
	topic := "mytopic/test"
	goldMqttResponse := "mymessage"
	MqttRequest := "mymessagee"

	mqtt := NewMqttTestClient()
	defer mqtt.Close()

	mqtt.MqttSendAndReceiveSync(t, topic, MqttRequest, goldMqttResponse)
}
