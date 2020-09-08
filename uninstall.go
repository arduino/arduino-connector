package main

import mqtt "github.com/eclipse/paho.mqtt.golang"

// Uninstall remove all service installed and create script to
// removing application from machine
func (s *Status) Uninstall(client mqtt.Client, msg mqtt.Message) {
	data := "OK"
	s.SendInfo(s.topicPertinence+"/status/uninstall", string(data))
}
