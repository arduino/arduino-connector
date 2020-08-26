package main

import (
	"log"
	"strings"
	"testing"
	"time"

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
