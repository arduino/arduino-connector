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
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/eclipse/paho.mqtt.golang"
)

// ExecAsVagrantSshCmd "wraps vagrant ssh -c
func ExecAsVagrantSshCmd(command string) (string, error) {
	vagrantSSHCmd := fmt.Sprintf(`cd test && vagrant ssh -c "%s"`, command)
	cmd := exec.Command("bash", "-c", vagrantSSHCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// MqttTestClient is an ad-hoc mqtt client struct for test
type MqttTestClient struct {
	client        mqtt.Client
	thingToTestId string
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
	url := "endpoint.iot.com"
	for _, line := range strings.Split(string(file), "\n") {
		if strings.Contains(line, "export IOT_ENDPOINT") {
			url = strings.Split(line, "=")[1]
		}
	}
	file, err = ioutil.ReadFile("test/ui_gen_install.sh")
	if err != nil {
		panic(err)
	}
	thingToTestId := "thing:id-id-id-id"
	for _, line := range strings.Split(string(file), "\n") {
		if strings.Contains(line, "export id") {
			thingToTestId = strings.Split(line, "=")[1]
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

	return &MqttTestClient{
		client,
		thingToTestId,
	}
}

func (tmc *MqttTestClient) Close() {
	tmc.client.Disconnect(100)
}

func (tmc *MqttTestClient) MqttSendAndReceiveSync(t *testing.T, topic, request string) string {

	iotTopic := strings.Join([]string{"$aws/things", tmc.thingToTestId, topic}, "/")
	var wg sync.WaitGroup
	wg.Add(1)
	response := "none"
	if token := tmc.client.Subscribe(iotTopic, 0, func(client mqtt.Client, msg mqtt.Message) {
		response = string(msg.Payload())
		wg.Done()
	}); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}

	postTopic := strings.Join([]string{iotTopic, "post"}, "/")
	if token := tmc.client.Publish(postTopic, 0, false, request); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}
	wg.Wait()
	return response

}

// tests
func TestSketchProcessIsRunning(t *testing.T) {
	mqtt := NewMqttTestClient()
	defer mqtt.Close()
	sketchTopic := "upload"

	sketchDownloadCommand := fmt.Sprintf(`{"token": "","url": "%s","name": "sketch_devops_integ_test","id": "0774e17e-f60e-4562-b87d-18017b6ef3d2"}`, os.Getenv("SKETCH_DEVOPS_INTEG_TEST_BIN"))
	responseSketchRun := mqtt.MqttSendAndReceiveSync(t, sketchTopic, sketchDownloadCommand)
	t.Log(responseSketchRun)
	// outputMessage, err := ExecAsVagrantSshCmd("systemctl status ArduinoConnector | grep running")
	//if err != nil {
	//	t.Error(err)
	//}
	//assert.Equal(t, true, strings.Contains(responseSketchRun, "INFO: Sketch started with PID"))
	assert.Equal(t, true, strings.Contains(responseSketchRun, "INFO: Sketch started with PID"))
}
