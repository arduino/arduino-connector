//
//  This file is part of arduino-connector
//
//  Copyright (C) 2017-2020  Arduino AG (http://www.arduino.cc/)
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

	apt "github.com/arduino/go-apt-client"
	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// AptRepositoryListEvent sends a list of available repositories
func (s *Status) AptRepositoryListEvent(client mqtt.Client, msg mqtt.Message) {
	all, err := apt.ParseAPTConfigFolder("/etc/apt")
	if err != nil {
		s.Error("/apt/repos/list", fmt.Errorf("Retrieving repositories: %s", err))
		return
	}

	data, err := json.Marshal(all)
	if err != nil {
		s.Error("/apt/repos/list", fmt.Errorf("Json marshal result: %s", err))
		return
	}

	s.SendInfo(s.topicPertinence+"/apt/repos/list", string(data))
}

// AptRepositoryAddEvent adds a repository to the apt configuration
func (s *Status) AptRepositoryAddEvent(client mqtt.Client, msg mqtt.Message) {
	var params struct {
		Repository *apt.Repository `json:"repository"`
	}
	err := json.Unmarshal(msg.Payload(), &params)
	if err != nil {
		s.Error("/apt/repos/add", fmt.Errorf("Unmarshal '%s': %s", msg.Payload(), err))
		return
	}

	err = apt.AddRepository(params.Repository, "/etc/apt")
	if err != nil {
		s.Error("/apt/repos/add", fmt.Errorf("Adding repository '%s': %s", msg.Payload(), err))
		return
	}

	s.SendInfo(s.topicPertinence+"/apt/repos/add", "OK")
}

// AptRepositoryRemoveEvent removes a repository from the apt configuration
func (s *Status) AptRepositoryRemoveEvent(client mqtt.Client, msg mqtt.Message) {
	var params struct {
		Repository *apt.Repository `json:"repository"`
	}
	err := json.Unmarshal(msg.Payload(), &params)
	if err != nil {
		s.Error("/apt/repos/remove", fmt.Errorf("Unmarshal '%s': %s", msg.Payload(), err))
		return
	}

	err = apt.RemoveRepository(params.Repository, "/etc/apt")
	if err != nil {
		s.Error("/apt/repos/remove", fmt.Errorf("Removing repository '%s': %s", msg.Payload(), err))
		return
	}

	s.SendInfo(s.topicPertinence+"/apt/repos/remove", "OK")
}

// AptRepositoryEditEvent modifies a repository definition in the apt configuration
func (s *Status) AptRepositoryEditEvent(client mqtt.Client, msg mqtt.Message) {
	var params struct {
		OldRepository *apt.Repository `json:"old_repository"`
		NewRepository *apt.Repository `json:"new_repository"`
	}
	err := json.Unmarshal(msg.Payload(), &params)
	if err != nil {
		s.Error("/apt/repos/edit", fmt.Errorf("Unmarshal '%s': %s", msg.Payload(), err))
		return
	}

	err = apt.EditRepository(params.OldRepository, params.NewRepository, "/etc/apt")
	if err != nil {
		s.Error("/apt/repos/edit", fmt.Errorf("Changing repository '%s': %s", msg.Payload(), err))
		return
	}

	s.SendInfo(s.topicPertinence+"/apt/repos/edit", "OK")
}
