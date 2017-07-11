package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bcmi-labs/arduino-connector/auth"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/kardianos/osext"
	"github.com/kardianos/service"
	"github.com/pkg/errors"
)

const (
	rsaBits    = 2048
	devicesAPI = "https://api.arduino.cc/devices/v1"
)

// Install creates the necessary certificates and configuration files and installs the program as a service
func install(s service.Service, config Config, token string) {
	// Request token
	var err error
	if token == "" {
		token, err = askCredentials()
		check(err, "AskCredentials")
	}

	// Create a private key
	fmt.Println("Generate private key")
	key, err := generateKey("P256")
	check(err, "generateKey")

	keyOut, err := os.OpenFile("certificate.key", os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	check(err, "openKeyFile")

	pem.Encode(keyOut, pemBlockForKey(key))
	err = keyOut.Close()
	check(err, "closeKeyFile")

	// Create a csr
	fmt.Println("Generate csr")
	csr, err := generateCsr(key)
	check(err, "generateCsr")

	// Request a certificate
	fmt.Println("Request certificate")
	pem, err := requestCert(config.ID, token, csr)
	check(err, "requestCert")

	err = ioutil.WriteFile("certificate.pem", []byte(pem), 0600)
	check(err, "writeCertFile")

	// Request URL
	fmt.Println("Request mqtt url")
	config.URL, err = requestURL(token)
	check(err, "requestURL")

	// Write the configuration
	fmt.Println("Write conf to arduino-connector.cfg")
	data := config.String()
	err = ioutil.WriteFile("arduino-connector.cfg", []byte(data), 0660)
	check(err, "WriteConf")

	// InstallService
	err = s.Install()
	check(err, "InstallService")

	// Connect to MQTT and communicate back
	fmt.Println("Check successful mqtt connection")
	client, err := setupMQTTConnection("certificate.pem", "certificate.key", config.ID, config.URL)
	check(err, "ConnectMQTT")

	err = registerDevice(client, config.ID)
	check(err, "RegisterDevice")

	client.Disconnect(0)

	fmt.Println("Setup completed")
}

func askCredentials() (token string, err error) {
	var user, pass string
	fmt.Println("Insert your arduino username")
	fmt.Scanln(&user)
	fmt.Println("Insert your arduino password")
	fmt.Scanln(&pass)

	auth := auth.New()
	auth.ClientID = "connector"
	auth.Scopes = "iot:devices"
	tok, err := auth.Token(user, pass)
	if err != nil {
		return "", err
	}

	return tok.Access, nil
}

func generateKey(ecdsaCurve string) (interface{}, error) {
	switch ecdsaCurve {
	case "":
		return rsa.GenerateKey(rand.Reader, rsaBits)
	case "P224":
		return ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	case "P256":
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "P384":
		return ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	case "P521":
		return ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	default:
		return nil, fmt.Errorf("Unrecognized elliptic curve: %q", ecdsaCurve)
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}

var oidEmailAddress = asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 1}

func generateCsr(priv interface{}) ([]byte, error) {
	emailAddress := "test@example.com"
	subj := pkix.Name{
		CommonName:         "example.com",
		Country:            []string{"AU"},
		Province:           []string{"Some-State"},
		Locality:           []string{"MyCity"},
		Organization:       []string{"Company Ltd"},
		OrganizationalUnit: []string{"IT"},
	}
	rawSubj := subj.ToRDNSequence()
	rawSubj = append(rawSubj, []pkix.AttributeTypeAndValue{
		{Type: oidEmailAddress, Value: emailAddress},
	})
	asn1Subj, err := asn1.Marshal(rawSubj)
	if err != nil {
		return nil, err
	}
	template := x509.CertificateRequest{
		RawSubject:         asn1Subj,
		EmailAddresses:     []string{emailAddress},
		SignatureAlgorithm: x509.ECDSAWithSHA256,
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &template, priv)
	if err != nil {
		return nil, err
	}
	return csr, nil
}

func requestCert(id, token string, csr []byte) (string, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	pemData := bytes.NewBuffer([]byte{})
	pem.Encode(pemData, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})
	payload := `{"csr":"` + pemData.String() + `"}`
	payload = strings.Replace(payload, "\n", "\\n", -1)

	req, err := http.NewRequest("POST", devicesAPI+"/"+id, strings.NewReader(payload))
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", "Bearer "+token)

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", errors.New("POST " + "/" + id + ": expected 200 OK, got " + res.Status)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	data := struct {
		Certificate string `json:"certificate"`
	}{}

	err = json.Unmarshal(body, &data)

	return data.Certificate, nil
}

func requestURL(token string) (string, error) {
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequest("POST", devicesAPI+"/connect", nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", "Bearer "+token)

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

	data := struct {
		URL string `json:"url"`
	}{}

	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}

	return data.URL, nil
}

type program struct {
	Config Config
}

// Start run the program asynchronously
func (p *program) Start(s service.Service) error {
	go p.run()
	return nil
}

// Start run the program asynchronously
func (p *program) Stop(s service.Service) error {
	return nil
}

// createService returns the servcie to be installed
func createService(config Config) (service.Service, error) {
	workingDirectory, _ := osext.ExecutableFolder()

	svcConfig := &service.Config{
		Name:             "ArduinoConnector",
		DisplayName:      "Arduino Connector Service",
		Description:      "Cloud connector and launcher for Intel IoT devices.",
		Arguments:        []string{"-config", configFile},
		WorkingDirectory: workingDirectory,
	}

	prg := &program{config}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// registerDevice publishes on the topic /register with info about the device itself
func registerDevice(client mqtt.Client, id string) error {
	// get host
	host, err := os.Hostname()
	if err != nil {
		return err
	}

	// get Macs
	macs, err := getMACs()

	data := struct {
		Host string
		MACs []string
	}{
		Host: host,
		MACs: macs,
	}
	msg, err := json.Marshal(data)
	if err != nil {
		return err
	}

	if token := client.Publish("$aws/things/"+id+"/register", 1, false, msg); token.Wait() && token.Error() != nil {
		return err
	}

	return nil
}

// getMACs returns a list of MAC addresses found on the device
func getMACs() ([]string, error) {
	var macAddresses []string
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, errors.Wrap(err, "get net interfaces")
	}
	for _, netInterface := range interfaces {
		macAddress := netInterface.HardwareAddr
		hwAddr, err := net.ParseMAC(macAddress.String())
		if err != nil {
			continue
		}
		macAddresses = append(macAddresses, hwAddr.String())
	}
	return macAddresses, nil
}
