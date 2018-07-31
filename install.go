//
//  This file is part of arduino-connector
//
//  Copyright (C) 2017-2018  Arduino AG (http://www.arduino.cc/)
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
	"syscall"
	"time"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/arduino/arduino-connector/auth"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/facchinm/service"
	"github.com/kardianos/osext"
	"github.com/pkg/errors"
)

const (
	rsaBits = 2048
)

// Install installs the program as a service
func install(s service.Service) {
	// InstallService
	err := s.Install()
	// TODO: implement a fallback strtegy if service installation fails
	check(err, "InstallService")
}

// Register creates the necessary certificates and configuration files
func register(config Config, token string) {
	// Request token
	var err error
	if token == "" {
		token, err = askCredentials(config.AuthURL)
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
	csr, err := generateCsr(config.ID, key)
	check(err, "generateCsr")

	// Request a certificate
	fmt.Println("Request certificate")
	pem, err := requestCert(config.APIURL, config.ID, token, csr)
	check(err, "requestCert")

	err = ioutil.WriteFile("certificate.pem", []byte(pem), 0600)
	check(err, "writeCertFile")

	// Request URL
	fmt.Println("Request mqtt url")
	config.URL, err = requestURL(config.APIURL, token)
	check(err, "requestURL")

	// Write the configuration
	fmt.Println("Write conf to arduino-connector.cfg")
	data := config.String()
	err = ioutil.WriteFile("arduino-connector.cfg", []byte(data), 0660)
	check(err, "WriteConf")

	// Connect to MQTT and communicate back
	fmt.Println("Check successful mqtt connection")
	client, err := setupMQTTConnection("certificate.pem", "certificate.key", config.ID, config.URL, nil)
	check(err, "ConnectMQTT")

	err = registerDevice(client, config.ID)
	check(err, "RegisterDevice")

	client.Disconnect(0)

	fmt.Println("Setup completed")
}

func askCredentials(authURL string) (token string, err error) {
	var user, pass string
	fmt.Println("Insert your arduino username")
	fmt.Scanln(&user)
	fmt.Println("Insert your arduino password")

	bytePassword, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return "", err
	}
	pass = string(bytePassword)

	authClient := auth.New()
	authClient.CodeURL = authURL + "/oauth2/auth"
	authClient.TokenURL = authURL + "/oauth2/token"
	authClient.ClientID = "connector"
	authClient.Scopes = "iot:devices"

	var tok *auth.Token
	// Handle captcha
	for {
		tok, err = authClient.Token(user, pass)
		if err == nil || !strings.HasPrefix(err.Error(), "authenticate: CAPTCHA") {
			break
		}
		fmt.Println("The authentication requested a captcha! We can't let you solve it in a terminal, so please visit https://auth.arduino.cc/login. When you managed to log in from the browser come back here and press [Enter]")
		var temp string
		fmt.Scanln(&temp)
	}
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

func generateCsr(id string, priv interface{}) ([]byte, error) {
	emailAddress := id + "@arduino.cc"
	subj := pkix.Name{
		CommonName:         id,
		Country:            []string{"IT"},
		Province:           []string{"Piemonte"},
		Locality:           []string{"Torino"},
		Organization:       []string{"Arduino AG"},
		OrganizationalUnit: []string{"Cloud"},
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

func requestCert(apiURL, id, token string, csr []byte) (string, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	pemData := bytes.NewBuffer([]byte{})
	pem.Encode(pemData, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})
	payload := `{"csr":"` + pemData.String() + `"}`
	payload = strings.Replace(payload, "\n", "\\n", -1)

	req, err := http.NewRequest("POST", apiURL+"/iot/v1/devices/"+id, strings.NewReader(payload))
	if err != nil {
		return "", err
	}

	req.Header.Add("Authorization", "Bearer "+token)

	res, err := client.Do(req)
	if err != nil {
		return "", err
	}

	if res.StatusCode != 200 {
		return "", errors.New("POST " + apiURL + "/iot/v1/devices/" + id + ": expected 200 OK, got " + res.Status)
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

func requestURL(apiURL, token string) (string, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest("POST", apiURL+"/iot/v1/devices/connect", nil)
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
		return "", errors.Wrap(err, "unmarshal "+string(body))
	}

	return data.URL, nil
}

type program struct {
	Config     Config
	listenFile string
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
func createService(config Config, listenFile string) (service.Service, error) {
	workingDirectory, _ := osext.ExecutableFolder()

	svcConfig := &service.Config{
		Name:             "ArduinoConnector",
		DisplayName:      "Arduino Connector Service",
		Description:      "Cloud connector and launcher for Intel IoT devices.",
		Arguments:        []string{"-config", configFile},
		WorkingDirectory: workingDirectory,
		Dependencies:     []string{"network-online.target"},
	}

	prg := &program{config, listenFile}
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
