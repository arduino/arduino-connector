package main

import (
	"encoding/json"
	"log"
	"strings"
	"testing"
	"time"

	apt "github.com/arduino/go-apt-client"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
)

func TestAptList(t *testing.T) {
	ui := NewMqttTestClientLocal()
	defer ui.Close()

	s := NewStatus(program{}.Config, nil, nil, "")
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/apt/repos/list/post", s, s.AptRepositoryListEvent, false)

	resp := ui.MqttSendAndReceiveTimeout(t, "/apt/repos/list", "{}", 1*time.Second)

	if resp == "" {
		t.Error("response is empty")
	}
}

func TestAptAddError(t *testing.T) {
	ui := NewMqttTestClientLocal()
	defer ui.Close()

	s := NewStatus(program{}.Config, nil, nil, "")
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/apt/repos/add/post", s, s.AptRepositoryAddEvent, true)

	resp := ui.MqttSendAndReceiveTimeout(t, "/apt/repos/add", "{test}", 1*time.Second)

	assert.True(t, strings.HasPrefix(resp, "ERROR"))
	assert.True(t, strings.Contains(resp, "Unmarshal"))
}

func TestAptAdd(t *testing.T) {
	ui := NewMqttTestClientLocal()
	defer ui.Close()

	s := NewStatus(program{}.Config, nil, nil, "")
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/apt/repos/add/post", s, s.AptRepositoryAddEvent, true)

	var params struct {
		Repository *apt.Repository `json:"repository"`
	}

	params.Repository = &apt.Repository{
		URI: "www.test.io",
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Error(err)
	}

	resp := ui.MqttSendAndReceiveTimeout(t, "/apt/repos/add", string(data), 1*time.Second)
	assert.Equal(t, "INFO: OK\n", resp)

	// TODO: check if is it really added on list, I think apt.ParseAPTConfigFolder("/etc/apt") doens't read
	// all repository
}

func TestAptRemoveError(t *testing.T) {
	ui := NewMqttTestClientLocal()
	defer ui.Close()

	s := NewStatus(program{}.Config, nil, nil, "")
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/apt/repos/remove/post", s, s.AptRepositoryRemoveEvent, true)

	resp := ui.MqttSendAndReceiveTimeout(t, "/apt/repos/remove", "{test}", 1*time.Second)

	assert.True(t, strings.HasPrefix(resp, "ERROR"))
	assert.True(t, strings.Contains(resp, "Unmarshal"))
}

func TestAptRemove(t *testing.T) {
	ui := NewMqttTestClientLocal()
	defer ui.Close()

	s := NewStatus(program{}.Config, nil, nil, "")
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/apt/repos/remove/post", s, s.AptRepositoryRemoveEvent, true)

	var params struct {
		Repository *apt.Repository `json:"repository"`
	}

	params.Repository = &apt.Repository{
		URI: "www.test.io",
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Error(err)
	}

	resp := ui.MqttSendAndReceiveTimeout(t, "/apt/repos/remove", string(data), 1*time.Second)
	assert.Equal(t, "INFO: OK\n", resp)

	// TODO: check if is it really removed on list, I think apt.ParseAPTConfigFolder("/etc/apt") doens't read
	// all repository
}
