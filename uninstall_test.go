package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	docker "github.com/docker/docker/client"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
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

	file, errFile := os.Create(folder + "/sketches")
	if errFile != nil {
		t.Error(errFile)
	}

	_, err = file.WriteString("test")
	assert.True(t, err == nil)
	file.Close()

	_, err = os.Stat(folder + "/sketches")
	assert.True(t, err == nil)

	resp := dashboard.MqttSendAndReceiveTimeout(t, "/status/uninstall", "{}", 50*time.Millisecond)

	_, err = os.Stat(folder + "/sketches")

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

	_, err := file1.WriteString("test")
	assert.True(t, err == nil)
	_, err = file2.WriteString("test")
	assert.True(t, err == nil)
	file1.Close()
	file2.Close()

	_, err = os.Stat(s.config.CertPath + "/certificate.pem")
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

func TestUninstallGenerateScript(t *testing.T) {
	dashboard := newMqttTestClientLocal()
	defer dashboard.Close()

	s := NewStatus(Config{}, nil, nil, "")
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/status/uninstall/post", s, s.Uninstall, false)

	resp := dashboard.MqttSendAndReceiveTimeout(t, "/status/uninstall", "{}", 500*time.Millisecond)
	assert.True(t, resp == "INFO: OK\n")

	defer func() {
		err := os.Remove("/opt/uninstall-arduino-connector.sh")
		assert.True(t, err == nil)
	}()

	_, err := os.Stat("/opt/uninstall-arduino-connector.sh")
	assert.True(t, err == nil)
}

func TestUninstallDockerAllContainer(t *testing.T) {
	cli, err := docker.NewClientWithOpts(docker.WithVersion("1.38"))
	assert.True(t, err == nil)

	dashboard := newMqttTestClientLocal()
	defer dashboard.Close()

	s := NewStatus(Config{}, nil, nil, "")
	s.dockerClient = cli
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/status/uninstall/post", s, s.Uninstall, false)

	ctx := context.Background()
	reader, err := cli.ImagePull(ctx, "alpine", types.ImagePullOptions{})
	assert.True(t, err == nil)
	_, err = io.Copy(os.Stdout, reader)
	assert.True(t, err == nil)

	defer func() {
		_, err = s.dockerClient.ImageRemove(ctx, "alpine", types.ImageRemoveOptions{})
		assert.True(t, err == nil)
	}()

	err = createConfig()
	assert.True(t, err == nil)

	defer func() {
		err = os.RemoveAll(configDirectory)
		assert.True(t, err == nil)
	}()

	c, errCreate := cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine",
		Cmd:   []string{"echo", "hello world"},
	}, nil, nil, "myContainer")
	assert.True(t, errCreate == nil)

	updateConfigWithContainer(c.ID)

	resp := dashboard.MqttSendAndReceiveTimeout(t, "/status/uninstall", "{}", 5*time.Minute)
	assert.True(t, resp == "INFO: OK\n")
	containers, errList := cli.ContainerList(ctx, types.ContainerListOptions{})
	assert.True(t, errList == nil)
	found := false
	for _, v := range containers {
		if v.ID == c.ID {
			found = true
			break
		}
	}
	assert.False(t, found)
}

func TestUninstallNotAllDockerContainer(t *testing.T) {
	cli, err := docker.NewClientWithOpts(docker.WithVersion("1.38"))
	assert.True(t, err == nil)

	dashboard := newMqttTestClientLocal()
	defer dashboard.Close()

	s := NewStatus(Config{}, nil, nil, "")
	s.dockerClient = cli
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/status/uninstall/post", s, s.Uninstall, false)

	err = createConfig()
	assert.True(t, err == nil)

	defer func() {
		err = os.RemoveAll(configDirectory)
		assert.True(t, err == nil)
	}()

	ctx := context.Background()
	reader, err := cli.ImagePull(ctx, "alpine", types.ImagePullOptions{})
	assert.True(t, err == nil)
	_, err = io.Copy(os.Stdout, reader)
	assert.True(t, err == nil)
	defer func() {
		_, err = s.dockerClient.ImageRemove(ctx, "alpine", types.ImageRemoveOptions{})
		assert.True(t, err == nil)
	}()

	containerMustKeep, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine",
		Cmd:   []string{"echo", "hello world"},
	}, nil, nil, "")
	assert.True(t, err == nil)

	containerMustRemove, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "alpine",
		Cmd:   []string{"echo", "hello world"},
	}, nil, nil, "")

	assert.True(t, err == nil)
	updateConfigWithContainer(containerMustRemove.ID)

	defer func() {
		err = cli.ContainerRemove(ctx, containerMustKeep.ID, types.ContainerRemoveOptions{})
		assert.True(t, err == nil)
	}()

	resp := dashboard.MqttSendAndReceiveTimeout(t, "/status/uninstall", "{}", 5*time.Minute)
	assert.True(t, resp == "INFO: OK\n")
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	assert.True(t, err == nil)
	found := false
	for _, v := range containers {
		switch v.ID {
		case containerMustKeep.ID:
			found = true
		case containerMustRemove.ID:
			assert.Fail(t, "Container ID %s found, but should have been removed", containerMustRemove.ID)
		}
	}
	assert.True(t, found)
}

func TestUninstallAllImages(t *testing.T) {
	cli, err := docker.NewClientWithOpts(docker.WithVersion("1.38"))
	assert.True(t, err == nil)

	dashboard := newMqttTestClientLocal()
	defer dashboard.Close()

	s := NewStatus(Config{}, nil, nil, "")
	s.dockerClient = cli
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/status/uninstall/post", s, s.Uninstall, false)

	err = createConfig()
	assert.True(t, err == nil)

	defer func() {
		err = os.RemoveAll(configDirectory)
		assert.True(t, err == nil)
	}()

	ctx := context.Background()
	reader, err := cli.ImagePull(ctx, "docker.io/library/alpine", types.ImagePullOptions{})
	assert.True(t, err == nil)
	_, err = io.Copy(os.Stdout, reader)
	assert.True(t, err == nil)

	updateConfigWithImage("alpine")

	resp := dashboard.MqttSendAndReceiveTimeout(t, "/status/uninstall", "{}", 5*time.Minute)
	assert.True(t, resp == "INFO: OK\n")
	list, err := cli.ImageList(ctx, types.ImageListOptions{})
	assert.True(t, err == nil)
	found := false
	for _, v := range list {
		if v.ID == "alpine" {
			found = true
		}
	}
	assert.False(t, found)
}

func TestUninstallNetworkManagerNotRemove(t *testing.T) {
	dashboard := newMqttTestClientLocal()
	defer dashboard.Close()

	s := NewStatus(Config{}, nil, nil, "")
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/status/uninstall/post", s, s.Uninstall, false)

	c := exec.Command("bash", "-c", "apt-get install -y network-manager")
	_, errCmd := c.CombinedOutput()
	assert.True(t, errCmd == nil)

	defer func() {
		c := exec.Command("bash", "-c", "apt-get remove -y network-manager")
		_, err := c.CombinedOutput()
		assert.True(t, err == nil)
		assert.False(t, isNetManagerInstalled())
	}()

	errConfig := createConfig()
	assert.True(t, errConfig == nil)

	defer func() {
		err := os.RemoveAll(configDirectory)
		assert.True(t, err == nil)
	}()

	assert.True(t, isNetManagerInstalled())

	resp := dashboard.MqttSendAndReceiveTimeout(t, "/status/uninstall", "{}", 5*time.Minute)
	assert.True(t, resp == "INFO: OK\n")
	assert.True(t, isNetManagerInstalled())
}

func TestUninstallNetworkManager(t *testing.T) {
	dashboard := newMqttTestClientLocal()
	defer dashboard.Close()

	s := NewStatus(Config{}, nil, nil, "")
	s.mqttClient = mqtt.NewClient(mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("arduino-connector"))
	if token := s.mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}
	defer s.mqttClient.Disconnect(100)

	subscribeTopic(s.mqttClient, "0", "/status/uninstall/post", s, s.Uninstall, false)

	err := createConfig()
	assert.True(t, err == nil)

	defer func() {
		err = os.RemoveAll(configDirectory)
		assert.True(t, err == nil)
	}()

	assert.False(t, isNetManagerInstalled())

	c := exec.Command("bash", "-c", "apt-get install -y network-manager")
	_, err = c.CombinedOutput()
	assert.True(t, err == nil)

	assert.True(t, isNetManagerInstalled())

	resp := dashboard.MqttSendAndReceiveTimeout(t, "/status/uninstall", "{}", 5*time.Minute)
	assert.True(t, resp == "INFO: OK\n")

	assert.False(t, isNetManagerInstalled())
}
