package main

import (
	"log"
	"os"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
)

func TestUninstallSketches(t *testing.T) {
	dashboard := newMqttTestClientLocal()
	defer dashboard.Close()

	s := NewStatus(program{}.Config, nil, nil, "")
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/status/uninstall/post", s, s.Uninstall, false)

	s.config.SketchesPath = "/home"
	folder, err := getSketchFolder(s)
	if err != nil {
		t.Error(err)
	}

	file, errFile := os.Create(folder + "/fakeSketch")
	if errFile != nil {
		t.Error(errFile)
	}

	file.WriteString("test")
	file.Close()

	_, err = os.Stat(folder + "/fakeSketch")
	assert.True(t, err == nil)

	resp := dashboard.MqttSendAndReceiveTimeout(t, "/status/uninstall", "{}", 50*time.Millisecond)

	_, err = os.Stat(folder + "/fakeSketch")

	assert.True(t, resp == "INFO: OK\n")
	assert.True(t, err != nil)
}
