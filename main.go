package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	MQTT "github.com/eclipse/paho.mqtt.golang"
	uuid "github.com/satori/go.uuid"
)

type ConfigFile struct {
	Username string
	Token    string
}

type RegistrationInfo struct {
	Username string
	Token    string
	Uuid     string
	Host     string
	MACs     []string
}

type exposedFunctions struct {
	Name      string
	Arguments string
}

type sketchStatus struct {
	Name      string
	PID       int
	Status    string // could be bool if we don't allow Pause
	Endpoints []exposedFunctions
}

type StatusInfo struct {
	IP       []string
	Sketches []sketchStatus
}

func main() {
	// Set up channel on which to send signal notifications.
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, os.Kill)

	registering := false

	u1, err := getUUID()
	if err != nil {
		registering = true
	}
	fmt.Println("Using UID " + u1.String())

	config, err := readConfig("arduino_connector.config")
	if err != nil {
		os.Exit(1)
	}
	fmt.Printf("%+v", config)

	user := config[0].Username
	token := config[0].Token
	host, _ := os.Hostname()

	var regInfo RegistrationInfo
	regInfo.Host = host
	regInfo.MACs = getMACAddress()
	regInfo.Token = token
	regInfo.Username = user

	client, err := setupMQTTConnection(".", "ClientID", "myawsioturl.iot.us-west-2.amazonaws.com")
	if err != nil {
		os.Exit(2)
	}

	if registering {
		// publish our data (UUID, username and token) on /register endpoint
		client.Publish("/register", 1, true, regInfo)
	}

	// Subscribe to /upload endpoint
	client.Subscribe("/upload", 1, uploadCB)

	// Subscribe to /sketch endpoint
	// Sketches are identified by their name
	// The status should be retrieved by the NATS internal channel
	client.Subscribe("/sketch", 1, sketchCB)

	// loop forever until we get a KILL signal

	// Publish on /status endpoint
	// Status should contain : IP addresses, running processes, some diagnostic info

	go func() {
		// collect Status info
		var status StatusInfo
		status.IP = getIPAddress()
		// status.Sketches = something
		client.Publish("/status", 1, false, status)
		time.Sleep(5 * time.Second)
	}()

	// Wait for receiving a signal.
	<-sigc
}

func uploadCB(MQTT.Client, MQTT.Message) {

}

func sketchCB(MQTT.Client, MQTT.Message) {

}

func readConfig(configPath string) ([]ConfigFile, error) {
	// Read config file
	var config []ConfigFile
	b, err := ioutil.ReadFile("arduino_connector.conf")
	err = json.Unmarshal(b, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func setupMQTTConnection(certificateLocation, clientID, awsHost string) (MQTT.Client, error) {
	cer, err := tls.LoadX509KeyPair(filepath.Join(certificateLocation, "cert.pem"), filepath.Join(certificateLocation, "private.key"))
	if err != nil {
		return nil, err
	}

	cid := clientID

	// AutoReconnect option is true by default
	// CleanSession option is true by default
	// KeepAlive option is 30 seconds by default
	connOpts := MQTT.NewClientOptions() // This line is different, we use the constructor function instead of creating the instance ourselves.
	connOpts.SetClientID(cid)
	connOpts.SetMaxReconnectInterval(1 * time.Second)
	connOpts.SetTLSConfig(&tls.Config{Certificates: []tls.Certificate{cer}})

	host := awsHost
	port := 8883
	path := "/mqtt"

	brokerURL := fmt.Sprintf("tcps://%s:%d%s", host, port, path)
	connOpts.AddBroker(brokerURL)

	mqttClient := MQTT.NewClient(connOpts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}
	log.Println("[MQTT] Connected")

	return mqttClient, nil
}

func getUUID() (uuid.UUID, error) {
	var u1 uuid.UUID

	b, err := ioutil.ReadFile("uuid")
	if err != nil {
		u1 = uuid.NewV4()
		ioutil.WriteFile("uuid", []byte(u1.String()), 0600)
	} else {
		u1, _ = uuid.FromString(string(b))
	}
	return u1, err
}

func getIPAddress() []string {

	var ipAddresses []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		fmt.Println(err)
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				fmt.Println("Current IP address : ", ipnet.IP.String())
				ipAddresses = append(ipAddresses, ipnet.IP.String())
			}
		}
	}
	return ipAddresses
}

func getMACAddress() []string {

	var macAddresses []string
	interfaces, _ := net.Interfaces()
	for _, netInterface := range interfaces {

		//name := netInterface.Name
		macAddress := netInterface.HardwareAddr
		hwAddr, err := net.ParseMAC(macAddress.String())

		if err != nil {
			continue
		}
		macAddresses = append(macAddresses, hwAddr.String())
	}
	return macAddresses
}
