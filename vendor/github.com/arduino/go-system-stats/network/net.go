//
//  This file is part of go-system-stats library
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

package net

import (
	"github.com/arduino/gonetworkmanager"
)

// Stats contains data about the memory avaible/used
type Stats struct {
	Devices []gonetworkmanager.Device
	Status  string
}

// IPProxyConfig contains data about the proxy configuration
type IPProxyConfig struct {
	Manual bool
	Config gonetworkmanager.IpProxyConfig
}

// GetNetworkStats returns statistics about network
func GetNetworkStats() (*Stats, error) {
	nm, err := gonetworkmanager.NewNetworkManager()
	if err != nil {
		return nil, err
	}
	devices, err := nm.GetDevices()
	if err != nil {
		return nil, err
	}
	return &Stats{
		Devices: devices,
		Status:  nm.GetState().String(),
	}, nil
}

// AddWirelessConnection adds a WiFi connection
func AddWirelessConnection(ssid, password string) error {
	nm, err := gonetworkmanager.NewNetworkManager()
	if err != nil {
		return err
	}
	nm.AddWirelessConnection(ssid, password)
	return nil
}

// AddWiredConnection adds a wired connection
func AddWiredConnection(config IPProxyConfig) error {
	nm, err := gonetworkmanager.NewNetworkManager()
	if err != nil {
		return err
	}
	nm.AddWiredConnection(config.Manual, config.Config)
	return nil
}
