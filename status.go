package main

import (
	"encoding/json"
	"os"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
)

// Status contains info about the sketches running on the device
type Status struct {
	id         string
	mqttClient mqtt.Client
	Sketches   map[string]*SketchStatus `json:"sketches"`
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

	msg, err := json.Marshal(s)
	if err != nil {
		panic(err) // Means that something went really wrong
	}

	if token := s.mqttClient.Publish("/status", 1, false, msg); token.Wait() && token.Error() != nil {
		panic(err) // Means that something went really wrong
	}
}

// Error logs an error on the specified topic
func (s *Status) Error(topic string, err error) {
	token := s.mqttClient.Publish("$aws/things/"+s.id+topic, 1, false, "ERROR: "+err.Error()+"\n")
	token.Wait()
}

// Info logs a message on the specified topic
func (s *Status) Info(topic, msg string) {
	token := s.mqttClient.Publish("$aws/things/"+s.id+topic, 1, false, "INFO: "+msg+"\n")
	token.Wait()
}

// Info logs a message on the specified topic
func (s *Status) Raw(topic, msg string) {
	token := s.mqttClient.Publish("$aws/things/"+s.id+topic, 1, false, msg)
	token.Wait()
}

// Publish sens on the /status topic a json representation of the connector
func (s *Status) Publish() {
	data, err := json.Marshal(s)

	//var out bytes.Buffer
	//json.Indent(&out, data, "", "  ")
	//fmt.Println(string(out.Bytes()))

	if err != nil {
		s.Error("/status/error", errors.Wrap(err, "status request"))
		return
	}

	s.Info("/status", string(data)+"\n")
}
