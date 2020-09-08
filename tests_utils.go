package main

import mqtt "github.com/eclipse/paho.mqtt.golang"

func newMqttTestClientLocal() *MqttTestClient {
	uiOptions := mqtt.NewClientOptions().AddBroker("tcp://localhost:1883").SetClientID("UI")
	ui := mqtt.NewClient(uiOptions)
	if token := ui.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	return &MqttTestClient{
		ui,
		"",
	}
}
