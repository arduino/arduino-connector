package main

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"
	"testing"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

// MqttTestClient is an ad-hoc mqtt client struct for test
type MqttTestClient struct {
	client        mqtt.Client
	thingToTestID string
}

// NewMqttTestClient creates mqtt cliente with cert for aws
func NewMqttTestClient() *MqttTestClient {
	cert := "test/cert.pem"
	key := "test/privateKey.pem"
	id := "testThingVagrant"
	port := 8883
	path := "/mqtt"
	file, err := ioutil.ReadFile("test/cert_arn.sh")
	if err != nil {
		return nil
	}
	url := "endpoint.iot.com"
	for _, line := range strings.Split(string(file), "\n") {
		if strings.Contains(line, "export IOT_ENDPOINT") {
			url = strings.Split(line, "=")[1]
		}
	}
	file, err = ioutil.ReadFile("test/ui_gen_install.sh")
	if err != nil {
		return nil
	}
	thingToTestID := "thing:id-id-id-id"
	for _, line := range strings.Split(string(file), "\n") {
		if strings.Contains(line, "export id") {
			thingToTestID = strings.Split(line, "=")[1]
		}
	}
	brokerURL := fmt.Sprintf("tcps://%s:%d%s", url, port, path)
	cer, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil
	}
	opts := mqtt.NewClientOptions().AddBroker(brokerURL)
	opts.SetClientID(id)
	opts.SetTLSConfig(&tls.Config{
		Certificates: []tls.Certificate{cer},
		ServerName:   url,
	})

	client := mqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		return nil
	}

	return &MqttTestClient{
		client,
		thingToTestID,
	}
}

// Close disconnect client
func (tmc *MqttTestClient) Close() {
	tmc.client.Disconnect(100)
}

// MqttSendAndReceiveTimeout sends request to topic using timeout to return response
func (tmc *MqttTestClient) MqttSendAndReceiveTimeout(t *testing.T, topic, request string, timeout time.Duration) string {
	t.Helper()

	respChan := make(chan string)
	if token := tmc.client.Subscribe(topic, 0, func(client mqtt.Client, msg mqtt.Message) {
		respChan <- string(msg.Payload())
	}); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}

	postTopic := strings.Join([]string{topic, "post"}, "/")
	if token := tmc.client.Publish(postTopic, 0, false, request); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}

	select {
	case <-time.After(timeout):
		if token := tmc.client.Unsubscribe(topic); token.Wait() && token.Error() != nil {
			t.Fatal(token.Error())
		}
		close(respChan)

		t.Fatalf("MqttSendAndReceiveTimeout() timeout for topic %s", topic)

		return ""
	case resp := <-respChan:
		return resp
	}
}

// MqttSendAndReceiveSync sends request to topic and return response
func (tmc *MqttTestClient) MqttSendAndReceiveSync(t *testing.T, topic, request string) string {

	iotTopic := strings.Join([]string{"$aws/things", tmc.thingToTestID, topic}, "/")
	var wg sync.WaitGroup
	wg.Add(1)
	response := "none"
	if token := tmc.client.Subscribe(iotTopic, 0, func(client mqtt.Client, msg mqtt.Message) {
		response = string(msg.Payload())
		wg.Done()
	}); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}

	postTopic := strings.Join([]string{iotTopic, "post"}, "/")
	if token := tmc.client.Publish(postTopic, 0, false, request); token.Wait() && token.Error() != nil {
		t.Fatal(token.Error())
	}
	wg.Wait()
	return response
}
