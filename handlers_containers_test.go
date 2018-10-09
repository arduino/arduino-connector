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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

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
func TestConnectorProcessIsRunning(t *testing.T) {
	outputMessage, err := ExecAsVagrantSshCmd("systemctl status ArduinoConnector | grep running")
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, true, strings.Contains(outputMessage, "active (running)"))
}

func TestConnectorDockerIsRunning(t *testing.T) {
	outputMessage, err := ExecAsVagrantSshCmd("sudo docker version")
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, true, strings.Contains(outputMessage, "Version:"))
}

func TestContainersPs(t *testing.T) {
	topic := "containers/ps"
	goldMqttResponse := "INFO: []\n\n"
	MqttRequest := "{}"
	mqtt := NewMqttTestClient()
	defer mqtt.Close()

	response := mqtt.MqttSendAndReceiveSync(t, topic, MqttRequest)
	assert.Equal(t, goldMqttResponse, response)

	outputMessage, err := ExecAsVagrantSshCmd("sudo docker ps -a")
	if err != nil {
		t.Error(err)
	}
	t.Log(outputMessage)
	assert.Equal(t, 2, len(strings.Split(outputMessage, "\n")))

}

func TestContainersImages(t *testing.T) {
	topic := "containers/images"
	goldMqttResponse := "INFO: []\n\n"
	MqttRequest := "{}"
	mqtt := NewMqttTestClient()
	defer mqtt.Close()

	response := mqtt.MqttSendAndReceiveSync(t, topic, MqttRequest)
	assert.Equal(t, goldMqttResponse, response)

	outputMessage, err := ExecAsVagrantSshCmd("sudo docker images")
	if err != nil {
		t.Error(err)
	}
	t.Log(outputMessage)
	assert.Equal(t, 2, len(strings.Split(outputMessage, "\n")))
}

func TestContainersRunStopStartRemove(t *testing.T) {
	mqtt := NewMqttTestClient()
	defer mqtt.Close()

	topic := "containers/action"
	// container run both test mqtt response andon VM
	RunMqttRequest := `{
		"action": "run",
		"image": "redis",
		"name": "my-redis-container"
	  }`

	response := mqtt.MqttSendAndReceiveSync(t, topic, RunMqttRequest)
	t.Log(response)
	assert.Equal(t, false, strings.Contains(response, "ERROR: "))
	assert.Equal(t, true, strings.Contains(response, "my-redis-container"))
	assert.Equal(t, true, strings.Contains(response, "run"))
	isContainerNotReady := true
	outputMessage := ""
	var err error
	waitTimeoutInSecs := 10
	for isContainerNotReady {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", RunMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker ps -a | grep 'my-redis-container'")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotReady = !strings.Contains(outputMessage, "my-redis-container")
	}
	t.Log(outputMessage)
	assert.Equal(t, true, strings.Contains(outputMessage, "my-redis-container"))

	//container stop test
	containerID := strings.Split(outputMessage, " ")[0]
	StopMqttRequest := fmt.Sprintf(`{"action": "stop","id":"%s"}`, containerID)
	response = mqtt.MqttSendAndReceiveSync(t, topic, StopMqttRequest)
	t.Log(response)
	assert.Equal(t, false, strings.Contains(response, "ERROR: "))
	assert.Equal(t, true, strings.Contains(response, containerID))
	assert.Equal(t, true, strings.Contains(response, "stop"))
	isContainerNotStopped := true
	outputMessage = ""
	waitTimeoutInSecs = 10
	for isContainerNotStopped {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", StopMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker ps -a | grep 'my-redis-container'")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotStopped = !strings.Contains(outputMessage, "Exited ")
	}
	t.Log(outputMessage)
	assert.Equal(t, true, strings.Contains(outputMessage, "my-redis-container"))
	assert.Equal(t, true, strings.Contains(outputMessage, "Exited "))

	//container start test
	StartMqttRequest := fmt.Sprintf(`{"action": "start","id":"%s","background": true}`, containerID)
	response = mqtt.MqttSendAndReceiveSync(t, topic, StartMqttRequest)
	t.Log(response)
	assert.Equal(t, false, strings.Contains(response, "ERROR: "))
	assert.Equal(t, true, strings.Contains(response, containerID))
	assert.Equal(t, true, strings.Contains(response, "start"))
	isContainerNotStarted := true
	outputMessage = ""
	waitTimeoutInSecs = 10
	for isContainerNotStarted {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", StartMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker ps -a | grep 'my-redis-container'")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotStarted = !strings.Contains(outputMessage, "Up ")
	}
	t.Log(outputMessage)
	assert.Equal(t, true, strings.Contains(outputMessage, "my-redis-container"))
	assert.Equal(t, true, strings.Contains(outputMessage, "Up "))

	//container remove test
	RemoveMqttRequest := fmt.Sprintf(`{"action": "remove","id":"%s"}`, containerID)
	response = mqtt.MqttSendAndReceiveSync(t, topic, RemoveMqttRequest)
	t.Log(response)
	assert.Equal(t, false, strings.Contains(response, "ERROR: "))
	assert.Equal(t, true, strings.Contains(response, containerID))
	assert.Equal(t, true, strings.Contains(response, "remove"))
	isContainerNotStoppedBeforeRemoved := true
	outputMessage = ""
	waitTimeoutInSecs = 10
	for isContainerNotStoppedBeforeRemoved {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", RemoveMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker ps -a")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotStoppedBeforeRemoved = len(strings.Split(outputMessage, "\n")) > 2
	}
	t.Log(outputMessage)
	assert.Equal(t, 2, len(strings.Split(outputMessage, "\n")))

	isContainerNotRemoved := true
	outputMessage = ""
	waitTimeoutInSecs = 10
	for isContainerNotRemoved {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", RemoveMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker images")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotRemoved = len(strings.Split(outputMessage, "\n")) > 2
	}
	t.Log(outputMessage)
	assert.Equal(t, 2, len(strings.Split(outputMessage, "\n")))

}

func TestContainersRunWithAuthSaveAndRemove(t *testing.T) {
	mqtt := NewMqttTestClient()
	defer mqtt.Close()

	topic := "containers/action"
	// container run both test mqtt response andon VM
	RunMqttRequest := fmt.Sprintf(`{
		"action": "run",
		"save_registry_credentials":true,
		"image": "%s",
		"user": "%s",
		"password":"%s",
		"name": "my-private-img"
	  }`, os.Getenv("CONNECTOR_PRIV_IMAGE"), os.Getenv("CONNECTOR_PRIV_USER"), os.Getenv("CONNECTOR_PRIV_PASS"))

	response := mqtt.MqttSendAndReceiveSync(t, topic, RunMqttRequest)
	t.Log(response)
	assert.Equal(t, false, strings.Contains(response, "ERROR: "))
	assert.Equal(t, true, strings.Contains(response, "my-private-img"))
	assert.Equal(t, true, strings.Contains(response, "run"))
	isContainerNotReady := true
	outputMessage := ""
	var err error
	waitTimeoutInSecs := 10
	for isContainerNotReady {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", RunMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker ps -a | grep 'my-private-img'")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotReady = !strings.Contains(outputMessage, "my-private-img")
	}
	t.Log(outputMessage)
	assert.Equal(t, true, strings.Contains(outputMessage, "my-private-img"))
	containerID := strings.Split(outputMessage, " ")[0]
	//container remove test
	RemoveMqttRequest := fmt.Sprintf(`{"action": "remove","id":"%s"}`, containerID)
	response = mqtt.MqttSendAndReceiveSync(t, topic, RemoveMqttRequest)
	t.Log(response)
	assert.Equal(t, false, strings.Contains(response, "ERROR: "))
	assert.Equal(t, true, strings.Contains(response, containerID))
	assert.Equal(t, true, strings.Contains(response, "remove"))
	isContainerNotStoppedBeforeRemoved := true
	outputMessage = ""
	waitTimeoutInSecs = 10
	for isContainerNotStoppedBeforeRemoved {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", RemoveMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker ps -a")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotStoppedBeforeRemoved = len(strings.Split(outputMessage, "\n")) > 2
	}
	t.Log(outputMessage)
	assert.Equal(t, 2, len(strings.Split(outputMessage, "\n")))

	isContainerNotRemoved := true
	outputMessage = ""
	waitTimeoutInSecs = 10
	for isContainerNotRemoved {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", RemoveMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker images")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotRemoved = len(strings.Split(outputMessage, "\n")) > 2
	}
	t.Log(outputMessage)
	assert.Equal(t, 2, len(strings.Split(outputMessage, "\n")))

}

func TestContainersRunWithAuthSavedAndRemove(t *testing.T) {
	mqtt := NewMqttTestClient()
	defer mqtt.Close()

	topic := "containers/action"
	// container run both test mqtt response andon VM
	RunMqttRequest := fmt.Sprintf(`{
		"action": "run",
		"image": "%s",
		"name": "my-private-img"
	  }`, os.Getenv("CONNECTOR_PRIV_IMAGE"))

	response := mqtt.MqttSendAndReceiveSync(t, topic, RunMqttRequest)
	t.Log(response)
	assert.Equal(t, false, strings.Contains(response, "ERROR: "))
	assert.Equal(t, true, strings.Contains(response, "my-private-img"))
	assert.Equal(t, true, strings.Contains(response, "run"))
	isContainerNotReady := true
	outputMessage := ""
	var err error
	waitTimeoutInSecs := 10
	for isContainerNotReady {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", RunMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker ps -a | grep 'my-private-img'")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotReady = !strings.Contains(outputMessage, "my-private-img")
	}
	t.Log(outputMessage)
	assert.Equal(t, true, strings.Contains(outputMessage, "my-private-img"))
	containerID := strings.Split(outputMessage, " ")[0]
	//container remove test
	RemoveMqttRequest := fmt.Sprintf(`{"action": "remove","id":"%s"}`, containerID)
	response = mqtt.MqttSendAndReceiveSync(t, topic, RemoveMqttRequest)
	t.Log(response)
	assert.Equal(t, false, strings.Contains(response, "ERROR: "))
	assert.Equal(t, true, strings.Contains(response, containerID))
	assert.Equal(t, true, strings.Contains(response, "remove"))
	isContainerNotStoppedBeforeRemoved := true
	outputMessage = ""
	waitTimeoutInSecs = 10
	for isContainerNotStoppedBeforeRemoved {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", RemoveMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker ps -a")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotStoppedBeforeRemoved = len(strings.Split(outputMessage, "\n")) > 2
	}
	t.Log(outputMessage)
	assert.Equal(t, 2, len(strings.Split(outputMessage, "\n")))

	isContainerNotRemoved := true
	outputMessage = ""
	waitTimeoutInSecs = 10
	for isContainerNotRemoved {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", RemoveMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker images")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotRemoved = len(strings.Split(outputMessage, "\n")) > 2
	}
	t.Log(outputMessage)
	assert.Equal(t, 2, len(strings.Split(outputMessage, "\n")))

}

func TestContainersRunWithAuthTestFail(t *testing.T) {
	mqtt := NewMqttTestClient()
	defer mqtt.Close()

	topic := "containers/action"
	// container run both test mqtt response andon VM
	RunMqttRequest := fmt.Sprintf(`{
		"action": "run",
		"image": "%s",
		"user": "%s",
		"password":"%s",
		"name": "my-private-img"
	  }`, os.Getenv("CONNECTOR_PRIV_IMAGE"), os.Getenv("CONNECTOR_PRIV_USER"), "MYWRONGPASSWORD")

	response := mqtt.MqttSendAndReceiveSync(t, topic, RunMqttRequest)
	t.Log(response)
	assert.Equal(t, true, strings.Contains(response, "ERROR: "))
	assert.Equal(t, true, strings.Contains(response, "auth test failed"))
}

func TestMultipleContainersRunWithPsFilterCheck(t *testing.T) {
	mqtt := NewMqttTestClient()
	defer mqtt.Close()

	topic := "containers/action"
	// container run both test mqtt response andon VM

	responseAlfa := mqtt.MqttSendAndReceiveSync(t, topic, `{"action": "run","image": "redis","name": "redis-alfa"}`)
	responseAlfa = strings.Replace(responseAlfa, "INFO: ", "", 1)
	t.Log(responseAlfa)
	alfaParams := RunPayload{}
	merr := json.Unmarshal([]byte(responseAlfa), &alfaParams)
	if merr != nil {
		t.Fatalf("Unmarshal error: %s", responseAlfa)

	}
	responseBeta := mqtt.MqttSendAndReceiveSync(t, topic, `{"action": "run","image": "redis","name": "redis-beta"}`)
	responseBeta = strings.Replace(responseBeta, "INFO: ", "", 1)
	t.Log(responseBeta)
	betaParams := RunPayload{}
	merr = json.Unmarshal([]byte(responseBeta), &betaParams)
	if merr != nil {
		t.Fatalf("Unmarshal error: %s", responseBeta)

	}
	areContainersNotReady := true
	outputMessage := ""
	var err error
	waitTimeoutInSecs := 10
	for areContainersNotReady {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for multiple docker containers")
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker ps -a")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		areContainersNotReady = !(strings.Contains(outputMessage, "redis-alfa") &&
			strings.Contains(outputMessage, "redis-beta"))
	}
	t.Log(outputMessage)
	//container ps with filter test
	psAlfaResponse := mqtt.MqttSendAndReceiveSync(t, "containers/ps", fmt.Sprintf(`{"id": "%s" }`, alfaParams.ContainerID))
	t.Log(psAlfaResponse)
	psBetaResponse := mqtt.MqttSendAndReceiveSync(t, "containers/ps", fmt.Sprintf(`{"id": "%s" }`, betaParams.ContainerID))
	t.Log(psAlfaResponse)

	assert.Equal(t, true, strings.Contains(psAlfaResponse, alfaParams.ContainerID))
	assert.Equal(t, false, strings.Contains(psAlfaResponse, betaParams.ContainerID))
	assert.Equal(t, true, strings.Contains(psBetaResponse, betaParams.ContainerID))
	assert.Equal(t, false, strings.Contains(psBetaResponse, alfaParams.ContainerID))

	// cleanup
	RemoveMqttRequest := fmt.Sprintf(`{"action": "remove","id":"%s"}`, alfaParams.ContainerID)
	mqtt.MqttSendAndReceiveSync(t, topic, RemoveMqttRequest)
	RemoveMqttRequest = fmt.Sprintf(`{"action": "remove","id":"%s"}`, betaParams.ContainerID)
	mqtt.MqttSendAndReceiveSync(t, topic, RemoveMqttRequest)

	isContainerNotRemoved := true
	outputMessage = ""
	waitTimeoutInSecs = 20
	for isContainerNotRemoved {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", RemoveMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker images")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotRemoved = len(strings.Split(outputMessage, "\n")) > 2
	}
	t.Log(outputMessage)
	assert.Equal(t, 2, len(strings.Split(outputMessage, "\n")))

}

func TestContainersRunWithRenameAndRemove(t *testing.T) {
	mqtt := NewMqttTestClient()
	defer mqtt.Close()

	topic := "containers/action"
	// container run both test mqtt response andon VM
	RunMqttRequest := fmt.Sprintf(`{
		"action": "run",
		"image": "redis",
		"name": "banana"
	  }`)

	response := mqtt.MqttSendAndReceiveSync(t, topic, RunMqttRequest)
	t.Log(response)
	assert.Equal(t, false, strings.Contains(response, "ERROR: "))
	assert.Equal(t, true, strings.Contains(response, "banana"))
	assert.Equal(t, true, strings.Contains(response, "run"))
	isContainerNotReady := true
	outputMessage := ""
	var err error
	waitTimeoutInSecs := 10
	for isContainerNotReady {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", RunMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker ps -a | grep 'banana'")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotReady = !strings.Contains(outputMessage, "banana")
	}
	t.Log(outputMessage)
	assert.Equal(t, true, strings.Contains(outputMessage, "banana"))
	containerID := strings.Split(outputMessage, " ")[0]

	// rename the container and test
	response = mqtt.MqttSendAndReceiveSync(t, "containers/rename",
		fmt.Sprintf(`{"id": "%s","name":"mango"}`, containerID))
	t.Log(response)
	outputMessage, err = ExecAsVagrantSshCmd("sudo docker ps -a | grep 'mango'")
	t.Log(outputMessage)
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, true, strings.Contains(outputMessage, "mango"))

	//container remove test
	RemoveMqttRequest := fmt.Sprintf(`{"action": "remove","id":"%s"}`, containerID)
	response = mqtt.MqttSendAndReceiveSync(t, topic, RemoveMqttRequest)
	t.Log(response)
	assert.Equal(t, false, strings.Contains(response, "ERROR: "))
	assert.Equal(t, true, strings.Contains(response, containerID))
	assert.Equal(t, true, strings.Contains(response, "remove"))

	isContainerNotRemoved := true
	outputMessage = ""
	waitTimeoutInSecs = 10
	for isContainerNotRemoved {
		if waitTimeoutInSecs--; waitTimeoutInSecs == 0 {
			t.Fatalf("timeout waiting for: %s", RemoveMqttRequest)
		}
		time.Sleep(time.Second)
		outputMessage, err = ExecAsVagrantSshCmd("sudo docker images")
		t.Log(outputMessage)
		if err != nil {
			t.Error(err)
		}
		isContainerNotRemoved = len(strings.Split(outputMessage, "\n")) > 2
	}
	t.Log(outputMessage)
	assert.Equal(t, 2, len(strings.Split(outputMessage, "\n")))

}
