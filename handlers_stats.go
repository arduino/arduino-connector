//
//  This file is part of arduino-connector
//
//  Copyright (C) 2017-2018  Arduino AG (http://www.arduino.cc/)
//
//  Licensed under the Apache License, Version 2.0 (the "License");
//  you may not use this file except in compliance with the License.
//  You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//  Unless required by applicable law or agreed to in writing, software
//  distributed under the License is distributed on an "AS IS" BASIS,
//  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//  See the License for the specific language governing permissions and
//  limitations under the License.
//

package main

import (
	"encoding/json"
	"fmt"
	"os/exec"

	apt "github.com/arduino/go-apt-client"
	"github.com/arduino/go-system-stats/disk"
	"github.com/arduino/go-system-stats/mem"
	net "github.com/arduino/go-system-stats/network"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

// WiFiEvent tries to connect to the specified wifi network
func (s *Status) WiFiEvent(client mqtt.Client, msg mqtt.Message) {
	// try registering a new wifi network
	var info struct {
		SSID     string `json:"ssid"`
		Password string `json:"password"`
	}
	err := json.Unmarshal(msg.Payload(), &info)
	if err != nil {
		s.Error("/wifi", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
		return
	}
	err = net.AddWirelessConnection(info.SSID, info.Password)
	if err != nil {
		return
	}
}

// EthEvent tries to change IP/Netmask/DNS configuration of the wired connection
func (s *Status) EthEvent(client mqtt.Client, msg mqtt.Message) {
	// try registering a new wifi network
	var info net.IPProxyConfig
	err := json.Unmarshal(msg.Payload(), &info)
	if err != nil {
		s.Error("/ethernet", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
		return
	}
	err = net.AddWiredConnection(info)
	if err != nil {
		return
	}
}

func checkAndInstallNetworkManager() {
	_, err := net.GetNetworkStats()
	if err == nil {
		return
	}

	dpkgCmd := exec.Command("dpkg", "--configure", "-a")
	if out, err := dpkgCmd.CombinedOutput(); err != nil {
		fmt.Println("Failed to dpkg configure all:")
		fmt.Println(string(out))
	}

	toInstall := &apt.Package{Name: "network-manager"}
	if out, err := apt.Install(toInstall); err != nil {
		fmt.Println("Failed to install network-manager:")
		fmt.Println(string(out))
		return
	}
	cmd := exec.Command("/etc/init.d/network-manager", "start")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Println("Failed to start network-manager:")
		fmt.Println(string(out))
	}
}

// StatsEvent sends statistics about resource used in the system (RAM, Disk, Network, etc...)
func (s *Status) StatsEvent(client mqtt.Client, msg mqtt.Message) {
	// Gather all system data metrics
	memStats, err := mem.GetStats()
	if err != nil {
		s.Error("/stats", fmt.Errorf("Retrieving memory stats: %s", err))
	}

	diskStats, err := disk.GetStats()
	if err != nil {
		s.Error("/stats", fmt.Errorf("Retrieving disk stats: %s", err))
	}

	netStats, err := net.GetNetworkStats()
	if err != nil {
		s.Error("/stats", fmt.Errorf("Retrieving network stats: %s", err))
	}

	type StatsPayload struct {
		Memory  *mem.Stats      `json:"memory"`
		Disk    []*disk.FSStats `json:"disk"`
		Network *net.Stats      `json:"network"`
	}

	info := StatsPayload{
		Memory:  memStats,
		Disk:    diskStats,
		Network: netStats,
	}

	// Send result
	data, err := json.Marshal(info)
	if err != nil {
		s.Error("/stats", fmt.Errorf("Json marsahl result: %s", err))
		return
	}

	//var out bytes.Buffer
	//json.Indent(&out, data, "", "  ")
	//fmt.Println(string(out.Bytes()))

	s.Info("/stats", string(data)+"\n")
}
