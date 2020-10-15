package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	apt "github.com/arduino/go-apt-client"
	"github.com/docker/docker/api/types"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

// Uninstall removes all service installed and creates script to
// removing application from machine
func (s *Status) Uninstall(client mqtt.Client, msg mqtt.Message) {
	data := "OK"

	if err := removeSketches(s); err != nil {
		panic(err)
	}

	if err := removeCerts(s); err != nil {
		panic(err)
	}

	if err := removeContainers(s); err != nil {
		panic(err)
	}

	if err := removeImages(s); err != nil {
		panic(err)
	}

	if err := removeNetworkManager(); err != nil {
		panic(err)
	}

	if err := generateScriptUninstall(); err != nil {
		panic(err)
	}

	s.SendInfo(s.topicPertinence+"/status/uninstall", string(data))
}

func removeSketches(s *Status) error {
	folder, err := getSketchFolder(s)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(folder); err != nil {
		return err
	}

	fmt.Println("return nil")
	return nil
}

func removeCerts(s *Status) error {
	pem := strings.Join([]string{s.config.CertPath, "certificate.pem"}, "/")

	if err := os.Remove(pem); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	key := strings.Join([]string{s.config.CertPath, "certificate.key"}, "/")
	if err := os.Remove(key); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	return nil
}

func removeContainers(s *Status) error {
	viper.SetConfigFile(filepath.Join(configDirectory, "arduino-connector.yml"))
	containers := viper.GetStringSlice("docker-container")
	if len(containers) == 0 {
		return nil
	}

	for _, v := range containers {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
		defer cancel()
		err := s.dockerClient.ContainerRemove(ctx, v, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			return err
		}
	}

	return nil
}

func removeImages(s *Status) error {
	viper.SetConfigFile(filepath.Join(configDirectory, "arduino-connector.yml"))
	images := viper.GetStringSlice("docker-images")
	if len(images) == 0 {
		return nil
	}

	for _, v := range images {
		_, err := s.dockerClient.ImageRemove(context.Background(), v, types.ImageRemoveOptions{})
		if err != nil {
			fmt.Println(err)
		}
	}

	return nil
}

func removeNetworkManager() error {
	viper.SetConfigFile(filepath.Join(configDirectory, "arduino-connector.yml"))
	before := viper.GetBool("network-manager-installed")
	now := isNetManagerInstalled()
	if !before && now {
		toRemove := append([]*apt.Package{}, &apt.Package{Name: "network-manager"})
		_, err := apt.Remove(toRemove...)
		if err != nil {
			return err
		}
	}

	return nil
}

func generateScriptUninstall() error {
	file, errFile := os.Create("/opt/uninstall-arduino-connector.sh")
	if errFile != nil {
		fmt.Println("error generating file in ", "/opt/uninstall-arduino-connector.sh")
		return errFile
	}

	defer file.Close()
	data := `sudo systemctl stop ArduinoConnector.service
sudo systemctl disable ArduinoConnector.service
sudo rm /etc/systemd/system/ArduinoConnector.service
sudo rm -rf /opt/arduino-connector
sudo systemctl daemon-reload
sudo systemctl reset-failed
sudo rm -f /usr/bin/arduino-connector`
	_, err := file.WriteString(data)
	return err
}
