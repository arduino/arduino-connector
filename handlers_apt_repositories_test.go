package main

import (
	"encoding/json"
	"fmt"
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

	assert.NotEmpty(t, resp)
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
		Enabled:      false,
		SourceRepo:   true,
		URI:          "http://ppa.launchpad.net/test/ubuntu",
		Distribution: "zesty",
		Components:   "main",
		Comment:      "",
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Error(err)
	}

	resp := ui.MqttSendAndReceiveTimeout(t, "/apt/repos/add", string(data), 1*time.Second)
	assert.Equal(t, "INFO: OK\n", resp)

	defer func() {
		err = apt.RemoveRepository(params.Repository, "/etc/apt")
		if err != nil {
			t.Error(err)
		}
	}()

	all, err := apt.ParseAPTConfigFolder("/etc/apt")
	if err != nil {
		s.Error("/apt/repos/list", fmt.Errorf("Retrieving repositories: %s", err))
		return
	}

	assert.True(t, all.Contains(params.Repository))
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
		Enabled:      false,
		SourceRepo:   true,
		URI:          "http://ppa.launchpad.net/test/ubuntu",
		Distribution: "zesty",
		Components:   "main",
		Comment:      "",
	}

	errAdd := apt.AddRepository(params.Repository, "/etc/apt")
	if errAdd != nil {
		t.Error(errAdd)
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Error(err)
	}

	resp := ui.MqttSendAndReceiveTimeout(t, "/apt/repos/remove", string(data), 1*time.Second)
	assert.Equal(t, "INFO: OK\n", resp)

	all, err := apt.ParseAPTConfigFolder("/etc/apt")
	if err != nil {
		s.Error("/apt/repos/list", fmt.Errorf("Retrieving repositories: %s", err))
		return
	}

	assert.False(t, all.Contains(params.Repository))
}

func TestAptEditError(t *testing.T) {
	ui := NewMqttTestClientLocal()
	defer ui.Close()

	s := NewStatus(program{}.Config, nil, nil, "")
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/apt/repos/edit/post", s, s.AptRepositoryEditEvent, true)

	resp := ui.MqttSendAndReceiveTimeout(t, "/apt/repos/edit", "{test}", 1*time.Second)

	assert.True(t, strings.HasPrefix(resp, "ERROR"))
	assert.True(t, strings.Contains(resp, "Unmarshal"))
}

func TestAptEdit(t *testing.T) {
	ui := NewMqttTestClientLocal()
	defer ui.Close()

	s := NewStatus(program{}.Config, nil, nil, "")
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/apt/repos/edit/post", s, s.AptRepositoryEditEvent, true)

	var params struct {
		OldRepository *apt.Repository `json:"old_repository"`
		NewRepository *apt.Repository `json:"new_repository"`
	}

	params.OldRepository = &apt.Repository{
		Enabled:      false,
		SourceRepo:   true,
		URI:          "http://ppa.launchpad.net/OldTest/ubuntu",
		Distribution: "zesty",
		Components:   "main",
		Comment:      "old",
	}

	params.NewRepository = &apt.Repository{
		Enabled:      false,
		SourceRepo:   true,
		URI:          "http://ppa.launchpad.net/NewTest/ubuntu",
		Distribution: "zesty",
		Components:   "main",
		Comment:      "new",
	}

	errAdd := apt.AddRepository(params.OldRepository, "/etc/apt")
	if errAdd != nil {
		t.Error(errAdd)
	}

	defer func() {
		err := apt.RemoveRepository(params.NewRepository, "/etc/apt")
		if err != nil {
			t.Error(err)
		}
	}()

	data, err := json.Marshal(params)
	if err != nil {
		t.Error(err)
	}

	resp := ui.MqttSendAndReceiveTimeout(t, "/apt/repos/edit", string(data), 1*time.Second)
	assert.Equal(t, "INFO: OK\n", resp)

	all, err := apt.ParseAPTConfigFolder("/etc/apt")
	if err != nil {
		s.Error("/apt/repos/list", fmt.Errorf("Retrieving repositories: %s", err))
		return
	}

	assert.True(t, all.Contains(params.NewRepository))
	assert.False(t, all.Contains(params.OldRepository))
}
