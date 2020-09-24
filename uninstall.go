package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/arduino/go-apt-client"

	"github.com/docker/docker/api/types"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/kardianos/osext"
	"github.com/spf13/viper"
	"golang.org/x/net/context"
)

// Uninstall removes all service installed and creates script to
// removing application from machine
func (s *Status) Uninstall(client mqtt.Client, msg mqtt.Message) {
	data := "OK"

	var err error

	err = removeSketches(s)
	if err != nil {
		panic(err)
	}

	err = removeCerts(s)
	if err != nil {
		panic(err)
	}

	err = removeContainers(s)
	if err != nil {
		panic(err)
	}

	err = removeImages(s)
	if err != nil {
		panic(err)
	}

	err = removeNetworkManager()
	if err != nil {
		panic(err)
	}

	err = generateScriptUninstall()
	if err != nil {
		panic(err)
	}

	s.SendInfo(s.topicPertinence+"/status/uninstall", string(data))
}

func removeSketches(s *Status) error {
	folder, err := getSketchFolder(s)
	if err != nil {
		return err
	}

	_, err = os.Stat(folder)
	if err != nil {
		return nil
	}

	err = os.RemoveAll(folder)
	if err != nil {
		return err
	}

	return nil
}

func removeCerts(s *Status) error {
	pem := strings.Join([]string{s.config.CertPath, "certificate.pem"}, "/")

	_, err := os.Stat(pem)
	if err != nil {
		return nil
	}

	err = os.Remove(pem)
	if err != nil {
		return err
	}

	key := strings.Join([]string{s.config.CertPath, "certificate.key"}, "/")
	_, err = os.Stat(key)
	if err != nil {
		return nil
	}

	err = os.Remove(key)
	if err != nil {
		return err
	}

	return nil
}

func removeContainers(s *Status) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	viper.SetConfigFile(dir + string(os.PathSeparator) + "arduino-connector.yml")
	containers := viper.GetStringSlice("docker-container")
	if len(containers) == 0 {
		return nil
	}

	for _, v := range containers {
		err = s.dockerClient.ContainerRemove(context.Background(), v, types.ContainerRemoveOptions{Force: true})
		time.Sleep(5 * time.Second)
		if err != nil {
			return err
		}
	}

	return nil
}

func removeImages(s *Status) error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	viper.SetConfigFile(dir + string(os.PathSeparator) + "arduino-connector.yml")
	images := viper.GetStringSlice("docker-images")
	if len(images) == 0 {
		return nil
	}

	for _, v := range images {
		_, err = s.dockerClient.ImageRemove(context.Background(), v, types.ImageRemoveOptions{})
		time.Sleep(5 * time.Second)
		if err != nil {
			fmt.Println(err)
		}
	}

	return nil
}

func removeNetworkManager() error {
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	viper.SetConfigFile(dir + string(os.PathSeparator) + "arduino-connector.yml")
	net := viper.GetBool("network-manager")
	if net {
		toRemove := append([]*apt.Package{}, &apt.Package{Name: "network-manager"})
		_, err = apt.Remove(toRemove...)
		if err != nil {
			return err
		}
	}

	return nil
}

func generateScriptUninstall() error {
	dir, errDir := osext.ExecutableFolder()
	if errDir != nil {
		return errDir
	}

	fmt.Println(dir)

	file, errFile := os.Create(dir + "/uninstall-arduino-connector.sh")
	if errFile != nil {
		return errFile
	}

	defer file.Close()
	data := ""
	data += "sudo systemctl stop ArduinoConnector.service\n"
	data += "sudo systemctl disable ArduinoConnector.service\n"
	data += "sudo rm /etc/systemd/system/ArduinoConnector.service\n"
	data += "sudo systemctl daemon-reload\n"
	data += "sudo systemctl reset-failed\n"
	data += "sudo rm -f arduino-connector\n"
	_, err := file.WriteString(data)
	return err
}
