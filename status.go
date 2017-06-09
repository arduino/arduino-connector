package main

import (
	"encoding/json"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// Status contains info about the sketches running on the device
type Status struct {
	id       string
	client   mqtt.Client
	Sketches map[string]SketchStatus
}

// SketchStatus contains info about a single running sketch
type SketchStatus struct {
	Name      string
	PID       int
	Status    string // could be bool if we don't allow Pause
	Endpoints []Endpoint
}

// Endpoint is an exposed function
type Endpoint struct {
	Name      string
	Arguments string
}

// NewStatus creates a new status that publishes on a topic
func NewStatus(id string, client mqtt.Client) *Status {
	return &Status{
		id:       id,
		client:   client,
		Sketches: map[string]SketchStatus{},
	}
}

// Set adds or modify a sketch
func (s *Status) Set(name string, sketch SketchStatus) {
	s.Sketches[name] = sketch

	msg, err := json.Marshal(s)
	if err != nil {
		panic(err) // Means that something went really wrong
	}

	if token := s.client.Publish("/status", 1, false, msg); token.Wait() && token.Error() != nil {
		panic(err) // Means that something went really wrong
	}
}

// Error logs an error on the specified topic
func (s *Status) Error(topic string, err error) {
	token := s.client.Publish("$aws/things/"+s.id+topic, 1, false, "ERROR: "+err.Error())
	token.Wait()
}

// Info logs a message on the specified topic
func (s *Status) Info(topic, msg string) {
	token := s.client.Publish("$aws/things/"+s.id+topic, 1, false, "INFO: "+msg)
	token.Wait()
}
