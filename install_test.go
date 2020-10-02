// +build register

package main

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arduino/arduino-connector/auth"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

const (
	ID           = "arduino-connector-test"
	AuthClientID = "test-1234567890"
	DeviceCode   = "device-1234567890"
	AccessToken  = "test-token"
	CertPath     = "test-certs"
	URL          = "localhost"
)

func TestInstallCheckCreateConfig(t *testing.T) {
	err := createConfigFolder()
	assert.True(t, err == nil)
	defer func() {
		os.RemoveAll("/etc/arduino-connector")
	}()
	_, err = os.Stat("/etc/arduino-connector")
	assert.True(t, err == nil)
}

func TestInstallRegister(t *testing.T) {

	// mock the OAuth server
	oauthTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/device/code" {
			assert.Equal(t, http.MethodPost, r.Method)

			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("content-type"))

			body, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(t, "client_id="+AuthClientID+"&audience=https://api.arduino.cc", string(body))

			dc := auth.DeviceCode{
				DeviceCode:              DeviceCode,
				UserCode:                "",
				VerificationURI:         "",
				ExpiresIn:               0,
				Interval:                0,
				VerificationURIComplete: "http://test-verification-uri.com",
			}
			data, err := json.Marshal(dc)
			assert.NoError(t, err)

			w.Write(data)
		} else if r.URL.Path == "/oauth/token" {
			assert.Equal(t, http.MethodPost, r.Method)

			assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("content-type"))

			body, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)
			assert.Equal(
				t,
				"grant_type=urn%3Aietf%3Aparams%3Aoauth%3Agrant-type%3Adevice_code&device_code="+DeviceCode+"&client_id="+AuthClientID,
				string(body),
			)

			token := struct {
				AccessToken string `json:"access_token"`
				ExpiresIn   int    `json:"expires_in"`
				TokenType   string `json:"token_type"`
			}{
				AccessToken: AccessToken,
				ExpiresIn:   0,
				TokenType:   "",
			}
			data, err := json.Marshal(token)
			assert.NoError(t, err)

			w.Write(data)
		} else {
			t.Fatalf("unexpected path for oauth test server: %s", r.URL.Path)
		}
	}))
	defer oauthTestServer.Close()

	// mock the AWS API server
	awsApiTestServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/iot/v1/devices/connect" {
			assert.Equal(t, http.MethodPost, r.Method)

			assert.Equal(t, "Bearer "+AccessToken, r.Header.Get("Authorization"))

			resp := struct {
				URL string `json:"url"`
			}{
				URL: URL,
			}
			data, err := json.Marshal(resp)
			assert.NoError(t, err)

			w.Write(data)
		} else if strings.HasPrefix(r.URL.Path, "/iot/v1/devices/") {
			assert.Equal(t, "/iot/v1/devices/"+ID, r.URL.Path)

			assert.Equal(t, http.MethodPost, r.Method)

			assert.Equal(t, "Bearer "+AccessToken, r.Header.Get("Authorization"))

			body, err := ioutil.ReadAll(r.Body)
			assert.NoError(t, err)

			payload := string(body)
			payload = strings.Replace(payload, "\\n", "\n", -1)

			csr := []byte(strings.TrimSuffix(strings.TrimPrefix(payload, `{"csr":"`), `"}`))

			pemBlock, _ := pem.Decode(csr)
			assert.NotNil(t, pemBlock)

			clientCsr, err := x509.ParseCertificateRequest(pemBlock.Bytes)
			assert.NoError(t, err)

			assert.NoError(t, clientCsr.CheckSignature())

			checkCsrRawSubject(t, clientCsr)

			clientCrt, err := crsToCrt(
				filepath.Join(CertPath, "test-ca.crt"),
				filepath.Join(CertPath, "test-ca.key"),
				clientCsr,
			)
			assert.NoError(t, err, "error while generating client certificate from certificate signing request")
			assert.NotNil(t, clientCrt, "generate client certificate is empty")

			resp := struct {
				Certificate string `json:"certificate"`
			}{
				Certificate: string(clientCrt),
			}
			data, err := json.Marshal(resp)
			assert.NoError(t, err)

			w.Write(data)
		} else {
			t.Fatalf("unexpected path for aws api test server: %s", r.URL.Path)
		}
	}))
	defer awsApiTestServer.Close()

	// instantiate an observer client to read messages published by arduino-connector
	client, err := connectTestClient(
		filepath.Join(CertPath, "test-client.crt"),
		filepath.Join(CertPath, "test-client.key"),
	)
	assert.NoError(t, err)

	// subscribe to register topic and wait for a message
	var wg sync.WaitGroup
	wg.Add(1)
	client.Subscribe("$aws/things/"+ID+"/register", 1, func(client mqtt.Client, msg mqtt.Message) {
		defer wg.Done()

		var data struct {
			Host string
			MACs []string
		}

		err := json.Unmarshal(msg.Payload(), &data)
		assert.NoError(t, err)

		host, err := os.Hostname()
		assert.NoError(t, err)

		macs, err := getMACs()
		assert.NoError(t, err)

		assert.Equal(t, host, data.Host)
		assert.Equal(t, macs, data.MACs)
	})

	defer client.Disconnect(100)

	// call register function to test it
	testConfig := Config{
		ID:           ID,
		AuthURL:      oauthTestServer.URL,
		AuthClientID: AuthClientID,
		APIURL:       awsApiTestServer.URL,
		CertPath:     CertPath,
	}

	register(testConfig, "config.test", "")

	// check generated config
	buf, err := ioutil.ReadFile("config.test")
	assert.NoError(t, err)

	config := strings.Replace(string(buf), "\r\n", "\n", -1)
	expectedConfig := fmt.Sprintf(`id=%s
url=%s
http_proxy=
https_proxy=
all_proxy=
authurl=%s
auth_client_id=%s
apiurl=%s
cert_path=%s
sketches_path=
check_ro_fs=false
env_vars_to_load=
`, ID, URL, oauthTestServer.URL, AuthClientID, awsApiTestServer.URL, CertPath)

	assert.Equal(t, expectedConfig, config)

	// wait for mqtt test client callback to verify device register info
	wg.Wait()
}

func checkCsrRawSubject(t *testing.T, csr *x509.CertificateRequest) {
	var rdnSeq pkix.RDNSequence
	rest, err := asn1.Unmarshal(csr.RawSubject, &rdnSeq)
	assert.NoError(t, err)
	assert.Len(t, rest, 0)

	var subj pkix.Name
	subj.FillFromRDNSequence(&rdnSeq)

	assert.Len(t, subj.Country, 1)
	assert.Equal(t, "IT", subj.Country[0])

	assert.Len(t, subj.Organization, 1)
	assert.Equal(t, "Arduino AG", subj.Organization[0])

	assert.Len(t, subj.OrganizationalUnit, 1)
	assert.Equal(t, "Cloud", subj.OrganizationalUnit[0])

	assert.Len(t, subj.Province, 1)
	assert.Equal(t, "Piemonte", subj.Province[0])

	assert.Len(t, subj.Locality, 1)
	assert.Equal(t, "Torino", subj.Locality[0])

	assert.Equal(t, ID, subj.CommonName)

	objIds := make(map[string]string)
	for _, oid := range subj.Names {
		objIds[oid.Type.String()] = oid.Value.(string)
	}
	oidEmailKey := asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 9, 1}.String()
	assert.Contains(t, objIds, oidEmailKey)
	assert.Equal(t, ID+"@arduino.cc", objIds[oidEmailKey])
}

func crsToCrt(caCrtPath, caKeyPath string, csr *x509.CertificateRequest) ([]byte, error) {
	// load CA certificate and key
	caCrtFile, err := ioutil.ReadFile(caCrtPath)
	if err != nil {
		return nil, err
	}
	pemBlock, _ := pem.Decode(caCrtFile)
	if pemBlock == nil {
		return nil, err
	}
	caCrt, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}

	caKeyFile, err := ioutil.ReadFile(caKeyPath)
	if err != nil {
		return nil, err
	}
	pemBlock, _ = pem.Decode(caKeyFile)
	if pemBlock == nil {
		return nil, err
	}

	caPrivateKey, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}

	// create client certificate template
	clientCrtTemplate := x509.Certificate{
		Signature:          csr.Signature,
		SignatureAlgorithm: csr.SignatureAlgorithm,

		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		PublicKey:          csr.PublicKey,

		SerialNumber: big.NewInt(2),
		Issuer:       caCrt.Subject,
		Subject:      csr.Subject,
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	// create client certificate from template and CA public key
	clientCRTRaw, err := x509.CreateCertificate(
		rand.Reader,
		&clientCrtTemplate,
		caCrt,
		csr.PublicKey,
		caPrivateKey,
	)
	if err != nil {
		return nil, err
	}

	var pemCrt bytes.Buffer
	pem.Encode(&pemCrt, &pem.Block{Type: "CERTIFICATE", Bytes: clientCRTRaw})

	return pemCrt.Bytes(), nil
}

func connectTestClient(crtPath, keyPath string) (mqtt.Client, error) {
	// Read client certificate
	cer, err := tls.LoadX509KeyPair(crtPath, keyPath)
	if err != nil {
		return nil, errors.Wrap(err, "test-client read certificate")
	}

	opts := mqtt.NewClientOptions()
	opts.SetClientID("test-client")
	opts.SetMaxReconnectInterval(20 * time.Second)
	opts.SetConnectTimeout(30 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetTLSConfig(&tls.Config{
		Certificates: []tls.Certificate{cer},
		ServerName:   "localhost",
	})
	opts.AddBroker("tcps://localhost:8883/mqtt")

	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return nil, errors.Wrap(token.Error(), "test-client connection to broker")
	}

	return mqttClient, nil
}

func TestInstallDocker(t *testing.T) {
	checkAndInstallDocker()
	installed, err := isDockerInstalled()
	assert.True(t, err == nil)
	assert.True(t, installed)
}
