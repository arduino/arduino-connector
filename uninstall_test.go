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

func TestUninstallCerts(t *testing.T) {
	dashboard := newMqttTestClientLocal()
	defer dashboard.Close()

	c := Config{
		CertPath: "/home/",
	}
	s := NewStatus(c, nil, nil, "")
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/status/uninstall/post", s, s.Uninstall, false)

	file1, errFile1 := os.Create(s.config.CertPath + "/certificate.pem")
	if errFile1 != nil {
		t.Error(errFile1)
	}

	file2, errFile2 := os.Create(s.config.CertPath + "/certificate.key")
	if errFile2 != nil {
		t.Error(errFile2)
	}

	file1.WriteString("test")
	file2.WriteString("test")
	file1.Close()
	file2.Close()

	_, err := os.Stat(s.config.CertPath + "/certificate.pem")
	assert.True(t, err == nil)
	_, err = os.Stat(s.config.CertPath + "/certificate.key")
	assert.True(t, err == nil)

	resp := dashboard.MqttSendAndReceiveTimeout(t, "/status/uninstall", "{}", 50*time.Millisecond)
	assert.True(t, resp == "INFO: OK\n")

	_, err = os.Stat(s.config.CertPath + "/certificate.pem")
	assert.True(t, err != nil)
	_, err = os.Stat(s.config.CertPath + "/certificate.key")
	assert.True(t, err != nil)
}
