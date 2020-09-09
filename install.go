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
	"bytes"
	"context"
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
	"path/filepath"
	"strings"
	"time"

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

func createConfigFolder() error {
	err := os.Mkdir("/etc/arduino-connector/", 0755)
	if err != nil {
		return err
	}

	return nil
}

// Register creates the necessary certificates and configuration files
func register(config Config, configFile, token string) {
	// Request token
	var err error
	if token == "" {
		token, err = deviceAuth(config.AuthURL, config.AuthClientID)
		check(err, "deviceAuth")
	}

	// Generate a Private Key and CSR
	csr := generateKeyAndCsr(config)

	// Request Certificate and service URL to iot service
	config = requestCertAndBrokerURL(csr, config, configFile, token)

	// Connect to MQTT and communicate back
	registerDeviceViaMQTT(config)

	fmt.Println("Setup completed")
}

func generateKeyAndCsr(config Config) []byte {
	// Create a private key
	certKeyPath := filepath.Join(config.CertPath, "certificate.key")
	fmt.Println("Generate private key to dump in: ", certKeyPath)
	key, err := generateKey("P256")
	check(err, "generateKey")

	keyOut, err := os.OpenFile(certKeyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	check(err, "openKeyFile")

	err = pem.Encode(keyOut, pemBlockForKey(key))
	if err != nil {
		fmt.Println(err)
		return []byte{}
	}

	err = keyOut.Close()
	check(err, "closeKeyFile")

	// Create a csr
	fmt.Println("Generate csr")
	csr, err := generateCsr(config.ID, key)
	check(err, "generateCsr")
	return csr
}

func requestCertAndBrokerURL(csr []byte, config Config, configFile, token string) Config {
	// Request a certificate
	certPemPath := filepath.Join(config.CertPath, "certificate.pem")
	fmt.Println("Request certificate to dump in: ", certPemPath)
	pem, err := requestCert(config.APIURL, config.ID, token, csr)
	check(err, "requestCert")

	err = ioutil.WriteFile(certPemPath, []byte(pem), 0600)
	check(err, "writeCertFile")

	// Request URL
	fmt.Println("Request mqtt url")
	config.URL, err = requestURL(config.APIURL, token)
	check(err, "requestURL")

	// Write the configuration
	fmt.Println("Write conf to ", configFile)
	data := config.String()
	err = ioutil.WriteFile(configFile, []byte(data), 0660)
	check(err, "WriteConf")

	return config
}

func registerDeviceViaMQTT(config Config) {
	// Connect to MQTT and communicate back
	certPemPath := filepath.Join(config.CertPath, "certificate.pem")
	certKeyPath := filepath.Join(config.CertPath, "certificate.key")

	fmt.Println("Check successful MQTT connection")
	client, err := setupMQTTConnection(certPemPath, certKeyPath, config.ID, config.URL, nil)
	check(err, "ConnectMQTT")

	err = registerDevice(client, config.ID)
	check(err, "RegisterDevice")

	client.Disconnect(100)
	fmt.Println("MQTT connection successful")

}

// Implements Auth0 device authentication flow: https://auth0.com/docs/flows/guides/device-auth/call-api-device-auth
func deviceAuth(authURL, clientID string) (token string, err error) {
	auth.Init()
	code, err := auth.StartDeviceAuth(authURL, clientID)
	if err != nil {
		return "", err
	}

	fmt.Printf("Go to %s and confirm authentication\n", code.VerificationURIComplete)

	ticker := time.NewTicker(10 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	// Loop until the user authenticated or the timeout hits
Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop
		case <-ticker.C:
			var err error
			token, err = auth.CheckDeviceAuth(authURL, clientID, code.DeviceCode)
			if err == nil {
				cancel()
			}
		}
	}

	ticker.Stop()
	cancel()

	return token, nil
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

func formatCSR(csr []byte) string {
	pemData := bytes.NewBuffer([]byte{})
	err := pem.Encode(pemData, &pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})
	if err != nil {
		fmt.Println(err)
		return ""
	}

	return pemData.String()
}

func requestCert(apiURL, id, token string, csr []byte) (string, error) {
	client := http.Client{
		Timeout: 30 * time.Second,
	}
	formattedCSR := formatCSR(csr)
	payload := `{"csr":"` + formattedCSR + `"}`
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
	if err != nil {
		fmt.Println(err)
		return "", err
	}

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
func createService(config Config, configFile, listenFile string) (service.Service, error) {
	workingDirectory, _ := osext.ExecutableFolder()

	svcConfig := &service.Config{
		Name:             "ArduinoConnector",
		DisplayName:      "Arduino Connector Service",
		Description:      "Cloud connector and launcher for IoT devices.",
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
	if err != nil {
		fmt.Println(err)
		return err
	}

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
