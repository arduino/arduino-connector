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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	mqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/stretchr/testify/assert"
)

func assertEqual(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
}

func TestDockerPsApi(t *testing.T) {
	topic := "/containers/ps"
	broker := "tcp://localhost:1883"

	uiOptions := mqtt.NewClientOptions().AddBroker(broker).SetClientID("UI")
	ans := make(chan string)
	msgRcvd := func(client mqtt.Client, msg mqtt.Message) {
		s := string(msg.Payload())
		if s != "{}" {
			ans <- s
		}
	}
	uiOptions.SetDefaultPublishHandler(msgRcvd)

	uiOptions.OnConnect = func(c mqtt.Client) {
		if token := c.Subscribe(topic, 0, nil); token.Wait() && token.Error() != nil {
			t.Error(token.Error())
		}
	}

	ui := mqtt.NewClient(uiOptions)

	if token := ui.Connect(); token.Wait() && token.Error() != nil {
		t.Error(token.Error())
	}

	var p program

	status := NewStatus(p.Config, nil, nil)
	status.dockerClient, _ = docker.NewClientWithOpts(docker.WithVersion("1.38"))

	acOptions := mqtt.NewClientOptions().AddBroker(broker).SetClientID("arduino-connector")
	status.mqttClient = mqtt.NewClient(acOptions)

	if token := status.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}

	subscribeTopic(status.mqttClient, p.Config.ID, topic, status, status.ContainersPsEvent, false)

	if token := ui.Publish(topic, 0, false, "{}"); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}

	goldMqttResponse := "INFO: []\n\n"
	assertEqual(t, <-ans, goldMqttResponse)

	if token := ui.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		t.Error(token.Error())
	}

	if token := status.mqttClient.Unsubscribe(topic); token.Wait() && token.Error() != nil {
		t.Error(token.Error())
	}

	ui.Disconnect(250)
	status.mqttClient.Disconnect(250)
}

func TestConnectorProcessIsRunning(t *testing.T) {
	outputMessage, err := ExecAsVagrantSshCmd("systemctl status ArduinoConnector | grep running")
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, true, strings.Contains(outputMessage, "active (running)"))
}

func TestConnectorDockerIsRunningPlusPruneAll(t *testing.T) {
	outputMessage, err := ExecAsVagrantSshCmd("sudo docker version")
	if err != nil {
		t.Error(err)
	}
	_, err = ExecAsVagrantSshCmd("sudo docker system prune -a -f")
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
	registryEndpoint := strings.Split(os.Getenv("CONNECTOR_PRIV_IMAGE"), "/")[0]
	response := mqtt.MqttSendAndReceiveSync(t, topic, RunMqttRequest)
	t.Log(response)
	outputMessage, err := ExecAsVagrantSshCmd("sudo cat /root/.docker/config.json")
	if err != nil {
		t.Error(err)
	}
	t.Log(outputMessage)
	assert.Equal(t, false, strings.Contains(outputMessage, registryEndpoint))
	assert.Equal(t, true, strings.Contains(response, "ERROR: "))
	assert.Equal(t, true, strings.Contains(response, "auth test failed"))
}

func TestMultipleContainersRunWithPsAndImageFilterCheck(t *testing.T) {
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
	responseBeta := mqtt.MqttSendAndReceiveSync(t, topic, `{"action": "run","image": "mongo","name": "mongo-beta"}`)
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
			strings.Contains(outputMessage, "mongo-beta"))
	}
	t.Log(outputMessage)
	//container ps with filter test
	psAlfaResponse := mqtt.MqttSendAndReceiveSync(t, "containers/ps", fmt.Sprintf(`{"id": "%s" }`, alfaParams.ContainerID))
	t.Log(psAlfaResponse)
	psBetaResponse := mqtt.MqttSendAndReceiveSync(t, "containers/ps", fmt.Sprintf(`{"id": "%s" }`, betaParams.ContainerID))
	t.Log(psAlfaResponse)
	psWrongIdResponse := mqtt.MqttSendAndReceiveSync(t, "containers/ps", `{"id": "NON EXISTENT CONTAINER ID" }`)

	assert.Equal(t, true, strings.Contains(psAlfaResponse, alfaParams.ContainerID))
	assert.Equal(t, false, strings.Contains(psAlfaResponse, betaParams.ContainerID))
	assert.Equal(t, true, strings.Contains(psBetaResponse, betaParams.ContainerID))
	assert.Equal(t, false, strings.Contains(psBetaResponse, alfaParams.ContainerID))
	assert.Equal(t, false, strings.Contains(psWrongIdResponse, alfaParams.ContainerID))
	assert.Equal(t, false, strings.Contains(psWrongIdResponse, betaParams.ContainerID))

	// test also for image filtering
	containers := make([]types.Container, 10)
	psAlfaResponse = strings.Replace(psAlfaResponse, "INFO: ", "", 1)
	merr = json.Unmarshal([]byte(psAlfaResponse), &containers)
	if merr != nil {
		t.Fatalf("Unmarshal error: %s", responseBeta)
	}
	imageAlfaResponse := mqtt.MqttSendAndReceiveSync(t, "containers/images", fmt.Sprintf(`{"name": "%s" }`, containers[0].Image))
	t.Log(imageAlfaResponse)
	assert.Equal(t, true, strings.Contains(imageAlfaResponse, "redis"))
	assert.Equal(t, false, strings.Contains(imageAlfaResponse, "mongo"))

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
