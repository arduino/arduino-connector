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
	"os"
	"strconv"
	"time"

	docker "github.com/docker/docker/client"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

// Status contains info about the sketches running on the device
type Status struct {
	config          Config
	id              string
	mqttClient      mqtt.Client
	dockerClient    docker.APIClient
	Sketches        map[string]*SketchStatus `json:"sketches"`
	messagesSent    int
	firstMessageAt  time.Time
	topicPertinence string
}

// SketchBinding represents a pair (SketchName,SketchId)
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
	pty       *os.File
}

// Endpoint is an exposed function
type Endpoint struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// NewStatus creates a new status that publishes on a topic
func NewStatus(config Config, mqttClient mqtt.Client, dockerClient docker.APIClient, topicPertinence string) *Status {
	return &Status{
		config:          config,
		id:              config.ID,
		mqttClient:      mqttClient,
		dockerClient:    dockerClient,
		Sketches:        map[string]*SketchStatus{},
		topicPertinence: topicPertinence,
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
	if debugMqtt {
		fmt.Println("MQTT OUT: /status", string(msg))
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
	if debugMqtt {
		fmt.Println("MQTT OUT: $aws/things/"+s.id+topic, "ERROR: "+err.Error()+"\n")
	}
}

// Info logs a message on the specified topic
func (s *Status) Info(topic, msg string) bool {
	if s.mqttClient == nil {
		return false
	}
	s.messagesSent++
	token := s.mqttClient.Publish("$aws/things/"+s.id+topic, 1, false, "INFO: "+msg+"\n")
	res := token.Wait()
	if debugMqtt {
		fmt.Println("MQTT OUT: $aws/things/"+s.id+topic, "INFO: "+msg+"\n")
	}
	return res
}

// SendInfo send information to a specific topic
func (s *Status) SendInfo(topic, msg string) {
	if s.mqttClient == nil {
		return
	}

	s.messagesSent++

	if token := s.mqttClient.Publish(topic, 0, false, "INFO: "+msg+"\n"); token.Wait() && token.Error() != nil {
		s.Error(topic, token.Error())
	}

	if debugMqtt {
		fmt.Println("MQTT OUT: "+topic, "INFO: "+msg+"\n")
	}
}

// Raw sends a message on the specified topic without further processing
func (s *Status) Raw(topic, msg string) {
	if s.mqttClient == nil {
		return
	}

	if s.messagesSent < 10 {
		// first 10 messages are virtually free
		s.firstMessageAt = time.Now()
	}

	if s.messagesSent > 1000 {
		// if started more than one day ago, reset the counter
		if time.Since(s.firstMessageAt) > 24*time.Hour {
			s.messagesSent = 0
		}

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
	if debugMqtt {
		fmt.Println("MQTT OUT: $aws/things/"+s.id+topic, string(msg))
	}
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
