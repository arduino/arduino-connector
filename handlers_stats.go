//
//  This file is part of arduino-connector
//
//  Copyright (C) 2017  Arduino AG (http://www.arduino.cc/)
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

	"github.com/arduino/go-system-stats/disk"
	"github.com/arduino/go-system-stats/mem"
	"github.com/arduino/go-system-stats/network"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

type WiFiPayload struct {
	SSID     string `json:"ssid"`
	Password string `json:"password"`
}

func WiFiCB(s *Status) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		// try registering a new wifi network
		var info WiFiPayload
		err := json.Unmarshal(msg.Payload(), &info)
		if err != nil {
			s.Error("/wifi", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
			return
		}
		net.AddWirelessConnection(info.SSID, info.Password)
	}
}

// StatsCB sends statistics about resource used in the system (RAM, Disk, Network, etc...)
func StatsCB(s *Status) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		// Gather all system data metrics
		memStats, err := mem.GetStats()
		if err != nil {
			s.Error("/stats/error", fmt.Errorf("Retrieving memory stats: %s", err))
			return
		}

		diskStats, err := disk.GetStats()
		if err != nil {
			s.Error("/stats/error", fmt.Errorf("Retrieving disk stats: %s", err))
			return
		}

		netStats, err := net.GetNetworkStats()
		if err != nil {
			s.Error("/stats/error", fmt.Errorf("Retrieving network stats: %s", err))
			return
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
			s.Error("/stats/error", fmt.Errorf("Json marsahl result: %s", err))
			return
		}

		//var out bytes.Buffer
		//json.Indent(&out, data, "", "  ")
		//fmt.Println(string(out.Bytes()))

		s.Info("/stats", string(data)+"\n")
	}
}
