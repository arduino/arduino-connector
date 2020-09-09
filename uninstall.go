package main

import (
	"fmt"
	"os"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/kardianos/osext"
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
	file.WriteString(data)

	return nil
}