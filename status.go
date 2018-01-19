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
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

// Status contains info about the sketches running on the device
type Status struct {
	id           string
	mqttClient   mqtt.Client
	Sketches     map[string]*SketchStatus `json:"sketches"`
	messagesSent int
}

// Status contains info about the sketches running on the device
type StatusTemp struct {
	id         string
	mqttClient mqtt.Client
	Sketches   map[string]SketchStatus `json:"sketches"`
}

func ExpandStatus(s *Status) *StatusTemp {
	var temp StatusTemp
	temp.id = s.id
	temp.mqttClient = s.mqttClient
	temp.Sketches = make(map[string]SketchStatus)
	for _, element := range s.Sketches {
		temp.Sketches[element.Name] = *element
	}
	return &temp
}

type SketchBinding struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

// SketchStatus contains info about a single running sketch
type SketchStatus struct {
	Name      string     `json:"name"`
	ID        string     `json:"id"`
	PID       int        `json:"pid"`
	Status    string     `json:"status"` // could be bool if we don't allow Pause
	Endpoints []Endpoint `json:"endpoints"`
	pty       *os.File   `json:"-"`
}

// Endpoint is an exposed function
type Endpoint struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// NewStatus creates a new status that publishes on a topic
func NewStatus(id string, mqttClient mqtt.Client) *Status {
	return &Status{
		id:         id,
		mqttClient: mqttClient,
		Sketches:   map[string]*SketchStatus{},
	}
}

// Set adds or modify a sketch
func (s *Status) Set(name string, sketch *SketchStatus) {
	s.Sketches[name] = sketch

	if s.mqttClient == nil {
		return
	}
	msg, err := json.Marshal(s)
	if err != nil {
		panic(err) // Means that something went really wrong
	}

	s.messagesSent++
	if token := s.mqttClient.Publish("/status", 1, false, msg); token.Wait() && token.Error() != nil {
		panic(err) // Means that something went really wrong
	}
}

// Error logs an error on the specified topic
func (s *Status) Error(topic string, err error) {
	if s.mqttClient == nil {
		return
	}
	s.messagesSent++
	token := s.mqttClient.Publish("$aws/things/"+s.id+topic, 1, false, "ERROR: "+err.Error()+"\n")
	token.Wait()
}

// Info logs a message on the specified topic
func (s *Status) Info(topic, msg string) {
	if s.mqttClient == nil {
		return
	}
	s.messagesSent++
	token := s.mqttClient.Publish("$aws/things/"+s.id+topic, 1, false, "INFO: "+msg+"\n")
	token.Wait()
}

// Info logs a message on the specified topic
func (s *Status) Raw(topic, msg string) {
	if s.mqttClient == nil {
		return
	}
	if s.messagesSent > 1000 {
		fmt.Println("rate limiting: " + strconv.Itoa(s.messagesSent))
		introducedDelay := time.Duration(s.messagesSent/1000) * time.Second
		if introducedDelay > 20*time.Second {
			introducedDelay = 20 * time.Second
		}
		time.Sleep(introducedDelay)
	}
	s.messagesSent++
	token := s.mqttClient.Publish("$aws/things/"+s.id+topic, 1, false, msg)
	token.Wait()
}

// InfoCommandOutput sends command output on the specified topic
func (s *Status) InfoCommandOutput(topic string, out []byte) {
	// Prepare response payload
	type response struct {
		Output string `json:"output"`
	}
	info := response{Output: string(out)}
	data, err := json.Marshal(info)
	if err != nil {
		s.Error(topic, fmt.Errorf("Json marshal result: %s", err))
		return
	}

	// Send result
	s.Info(topic, string(data)+"\n")
}

// Publish sens on the /status topic a json representation of the connector
func (s *Status) Publish() {
	data, err := json.Marshal(s)

	//var out bytes.Buffer
	//json.Indent(&out, data, "", "  ")
	//fmt.Println(string(out.Bytes()))

	if err != nil {
		s.Error("/status", errors.Wrap(err, "status request"))
		return
	}

	s.Info("/status", string(data)+"\n")
}
