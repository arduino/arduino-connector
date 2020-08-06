// +build functional

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	docker "github.com/docker/docker/client"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
)

// NewMqttTestClientLocal creates mqtt client in localhost:1883
func NewMqttTestClientLocal() *MqttTestClient {
	uiOptions := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("UI")
	ui := mqtt.NewClient(uiOptions)
	if token := ui.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	return &MqttTestClient{
		ui,
		"",
	}
}

type testStatus struct {
	appStatus *Status
	ui        *MqttTestClient
}

var ts testStatus

func TestMain(m *testing.M) {
	os.Exit(setupAndRun(m))
}

func setupAndRun(m *testing.M) int {
	ts.ui = NewMqttTestClientLocal()
	defer ts.ui.Close()

	ts.appStatus = NewStatus(program{}.Config, nil, nil, "")
	ts.appStatus.dockerClient, _ = docker.NewClientWithOpts(docker.WithVersion("1.38"))
	ts.appStatus.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))

	defer ts.appStatus.mqttClient.Disconnect(100)

	return m.Run()
}

func TestDockerPsApi(t *testing.T) {
	if token := ts.appStatus.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}

	subscribeTopic(ts.appStatus.mqttClient, "0", "/containers/ps/post", ts.appStatus, ts.appStatus.ContainersPsEvent, false)
	resp := ts.ui.MqttSendAndReceiveTimeout(t, "/containers/ps", "{}", 50*time.Millisecond)

	// ask Docker about containers effectively running
	cmd := exec.Command("bash", "-c", "docker ps -a")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(string(out), "\n")
	// Remove the first line (command output header) and the last line (empty line)
	lines = lines[1 : len(lines)-1]

	// Take json without INFO tag
	resp = strings.TrimPrefix(resp, "INFO: ")
	resp = strings.TrimSuffix(resp, "\n\n")
	var result []types.Container
	if err := json.Unmarshal([]byte(resp), &result); err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, len(result), len(lines))
	for i, line := range lines {
		containerId := strings.Fields(line)[0]
		assert.True(t, strings.HasPrefix(result[i].ID, containerId))
	}
}
