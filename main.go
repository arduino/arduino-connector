package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"

	"github.com/oleksandr/bonjour"
	uuid "github.com/satori/go.uuid"
	"github.com/yosssi/gmq/mqtt"
	"github.com/yosssi/gmq/mqtt/client"
)

func main() {
	// Set up channel on which to send signal notifications.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	// Create an MQTT Client.
	cli := client.New(&client.Options{
		// Define the processing of the error handler.
		ErrorHandler: func(err error) {
			fmt.Println(err)
		},
	})

	// Terminate the Client.
	defer cli.Terminate()

	//roots, _ := x509.SystemCertPool()
	roots := x509.NewCertPool()

	// Read the certificate file.
	b, err := ioutil.ReadFile("mosquitto.org.crt")
	if err != nil {
		panic(err)
	}

	if ok := roots.AppendCertsFromPEM(b); !ok {
		panic("failed to parse root certificate")
	}

	tlsConfig := &tls.Config{
		RootCAs: roots,
	}

	var u1 uuid.UUID
	registering := false

	b, err = ioutil.ReadFile("uuid")
	if err != nil {
		u1 = uuid.NewV4()
		ioutil.WriteFile("uuid", []byte(u1.String()), 0600)
		registering = true
	} else {
		u1, _ = uuid.FromString(string(b))
	}

	// Connect to the MQTT Server.
	err = cli.Connect(&client.ConnectOptions{
		Network:   "tcp",
		Address:   "test.mosquitto.org:8883", //"cloud.arduino.cc:8883",
		TLSConfig: tlsConfig,
		ClientID:  []byte(u1.String()),
	})
	if err != nil {
		fmt.Println("Unable to connect") //panic(err)
	}

	fmt.Println("Using UID " + u1.String())

	// Setup our service export
	host, _ := os.Hostname()
	info := []string{"Arduino Connector Gateway Service"}

	_, err = bonjour.Register(host, "_arduino._tcp", "", 5335, []string{"type=" + info[0], "app=test"}, nil)
	if err != nil {
		log.Fatalln(err.Error())
	}
	// Subscribe to topics.
	err = cli.Subscribe(&client.SubscribeOptions{
		SubReqs: []*client.SubReq{
			&client.SubReq{
				TopicFilter: []byte(u1.String() + "/update"),
				QoS:         mqtt.QoS0,
				// Define the processing of the message handler.
				Handler: func(topicName, message []byte) {
					fmt.Println(string(topicName), string(message))
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	if registering {

		// Publish a message.
		err = cli.Publish(&client.PublishOptions{
			QoS:       mqtt.QoS0,
			TopicName: []byte("register/" + u1.String()),
			Message:   []byte(strings.Join(getMACAddress(), ",")),
		})
		if err != nil {
			panic(err)
		}
	}

	// Wait for receiving a signal.
	<-sigc

	// Disconnect the Network Connection.
	if err := cli.Disconnect(); err != nil {
		panic(err)
	}
}

func getMACAddress() []string {

	//----------------------
	// Get the local machine IP address
	// https://www.socketloop.com/tutorials/golang-how-do-I-get-the-local-ip-non-loopback-address
	//----------------------

	var macAddresses []string

	addrs, err := net.InterfaceAddrs()

	if err != nil {
		fmt.Println(err)
	}

	for _, address := range addrs {

		// check the address type and if it is not a loopback the display it
		// = GET LOCAL IP ADDRESS
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				fmt.Println("Current IP address : ", ipnet.IP.String())
			}
		}
	}

	fmt.Println("------------------------------")
	fmt.Println("We want the interface name that has the current IP address")
	fmt.Println("MUST NOT be binded to 127.0.0.1 ")
	fmt.Println("------------------------------")

	// get all the system's or local machine's network interfaces

	interfaces, _ := net.Interfaces()
	for _, netInterface := range interfaces {

		name := netInterface.Name
		macAddress := netInterface.HardwareAddr

		fmt.Println("Hardware name : ", name)
		fmt.Println("MAC address : ", macAddress)

		// verify if the MAC address can be parsed properly
		hwAddr, err := net.ParseMAC(macAddress.String())

		if err != nil {
			fmt.Println("No able to parse MAC address : ", err)
			continue
		}

		fmt.Printf("Physical hardware address : %s \n", hwAddr.String())
		macAddresses = append(macAddresses, hwAddr.String())
	}
	return macAddresses
}
