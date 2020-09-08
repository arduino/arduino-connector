package main

import (
	"log"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
)

func TestUninstallAPI(t *testing.T) {
	dashboard := newMqttTestClientLocal()
	defer dashboard.Close()

	s := NewStatus(program{}.Config, nil, nil, "")
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/status/uninstall/post", s, s.Uninstall, false)
	resp := dashboard.MqttSendAndReceiveTimeout(t, "/status/uninstall", "{}", 50*time.Millisecond)

	assert.True(t, resp == "INFO: OK\n")
}
