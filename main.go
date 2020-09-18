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
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	docker "github.com/docker/docker/client"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
	"github.com/namsral/flag"
	"github.com/nats-io/gnatsd/logger"
	"github.com/nats-io/gnatsd/server"
	"github.com/nats-io/go-nats"

	"github.com/pkg/errors"
)

const (
	defaultConfigFile = "/etc/arduino-connector/arduino-connector.cfg"
)

var (
	version   = "0.0.0-dev"
	debugMqtt = false
)

// Config holds the configuration needed by the application
type Config struct {
	ID            string
	URL           string
	HTTPProxy     string
	HTTPSProxy    string
	ALLProxy      string
	AuthURL       string
	AuthClientID  string
	APIURL        string
	updateURL     string
	appName       string
	CertPath      string
	SketchesPath  string
	CheckRoFs     bool
	SignatureKey  string
	EnvVarsToLoad string
}

func (c Config) String() string {
	out := "id=" + c.ID + "\r\n"
	out += "url=" + c.URL + "\r\n"
	out += "http_proxy=" + c.HTTPProxy + "\r\n"
	out += "https_proxy=" + c.HTTPSProxy + "\r\n"
	out += "all_proxy=" + c.ALLProxy + "\r\n"
	out += "authurl=" + c.AuthURL + "\r\n"
	out += "auth_client_id=" + c.AuthClientID + "\r\n"
	out += "apiurl=" + c.APIURL + "\r\n"
	out += "cert_path=" + c.CertPath + "\r\n"
	out += "sketches_path=" + c.SketchesPath + "\r\n"
	out += "check_ro_fs=" + strconv.FormatBool(c.CheckRoFs) + "\r\n"
	out += "env_vars_to_load=" + c.EnvVarsToLoad + "\r\n"
	return out
}

func main() {
	fmt.Println("Version: " + version)

	// Read config
	config := Config{}

	var doLogin = flag.Bool("login", false, "Do the login and prints out a temporary token")
	var doInstall = flag.Bool("install", false, "Install as a service")
	var doRegister = flag.Bool("register", false, "Registers on the cloud")
	var doProvision = flag.Bool("provision", false, "Provision key and CSR for the device")
	var doConfigure = flag.Bool("configure", false, "Connect and register on the cloud")
	var listenFile = flag.String("listen", "", "Tail given file and report percentage")
	var token = flag.String("token", "", "an authentication token")
	flag.StringVar(&config.updateURL, "updateUrl", "http://downloads.arduino.cc/tools/feed/", "")
	flag.StringVar(&config.appName, "appName", "arduino-connector", "")

	var configFile = flag.String(flag.DefaultConfigFlagname, "", "path to config file")
	flag.StringVar(&config.CertPath, "cert_path", "/etc/arduino-connector/", "path to store certificates")
	flag.StringVar(&config.SketchesPath, "sketches_path", "", "path to store sketches")
	flag.StringVar(&config.ID, "id", "", "id of the thing in aws iot")
	flag.StringVar(&config.URL, "url", "", "url of the thing in aws iot")
	flag.StringVar(&config.HTTPProxy, "http_proxy", "", "URL of HTTP proxy to use")
	flag.StringVar(&config.HTTPSProxy, "https_proxy", "", "URL of HTTPS proxy to use")
	flag.StringVar(&config.ALLProxy, "all_proxy", "", "URL of SOCKS proxy to use")
	flag.StringVar(&config.AuthURL, "authurl", "https://login.arduino.cc", "Url of authentication server")
	flag.StringVar(&config.AuthClientID, "auth_client_id", "", "Authentication client ID")
	flag.StringVar(&config.APIURL, "apiurl", "https://api2.arduino.cc", "Url of api server")
	flag.BoolVar(&config.CheckRoFs, "check_ro_fs", false, "Check for Read Only file system and remount if necessary")
	flag.BoolVar(&debugMqtt, "debug-mqtt", false, "Output all received/sent messages")
	flag.StringVar(&config.SignatureKey, "signature_key", "-----BEGIN PUBLIC KEY-----\nMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvc0yZr1yUSen7qmE3cxF\nIE12rCksDnqR+Hp7o0nGi9123eCSFcJ7CkIRC8F+8JMhgI3zNqn4cUEn47I3RKD1\nZChPUCMiJCvbLbloxfdJrUi7gcSgUXrlKQStOKF5Iz7xv1M4XOP3JtjXLGo3EnJ1\npFgdWTOyoSrA8/w1rck4c/ISXZSinVAggPxmLwVEAAln6Itj6giIZHKvA2fL2o8z\nCeK057Lu8X6u2CG8tRWSQzVoKIQw/PKK6CNXCAy8vo4EkXudRutnEYHEJlPkVgPn\n2qP06GI+I+9zKE37iqj0k1/wFaCVXHXIvn06YrmjQw6I0dDj/60Wvi500FuRVpn9\ntwIDAQAB\n-----END PUBLIC KEY-----", "key for verifying sketch binary signature")
	flag.StringVar(&config.EnvVarsToLoad, "env_vars_to_load", "", "List of comma-separated Environment variables to load from system before launching sketches binaries")

	flag.Parse()

	if config.AuthURL == "https://login.oniudra.cc" {
		config.AuthClientID = "ks1R298bA8IQnG4p6dPlbdEIXF6Kt1Lu"
	}

	if config.AuthURL == "https://login.arduino.cc" {
		config.AuthClientID = "QGdLCWFA4uQdbRE2NOFhUI8bnXWMZhCK"
	}

	if *configFile == "" {
		*configFile = defaultConfigFile
	}

	fmt.Printf("current configuration: %+v\n", config)

	// Create service and install
	s, err := createService(config, *configFile, *listenFile)
	check(err, "CreateService")

	if *doLogin {
		token, errDev := deviceAuth(config.AuthURL, config.AuthClientID)
		if errDev != nil {
			fmt.Fprintln(os.Stderr, errDev)
			os.Exit(1)
		}

		fmt.Println("Access Token:", token)
		log.Println("Access Token:", token)
		os.Exit(0)
	}

	if *doRegister {
		err = createConfigFolder()
		if err != nil {
			panic(err)
		}
		register(config, *configFile, *token)
	}

	if *doProvision {
		csr := generateKeyAndCsr(config)
		formattedCSR := formatCSR(csr)
		formattedCSR = strings.Replace(formattedCSR, "\n", "\\n", -1)
		fmt.Println(formattedCSR)
		// provision should return cleanly if succeeded
		os.Exit(0)
	}

	// if configure flag is used the connector assumes that the config file is correctly written and the certificate.pem file is present
	if *doConfigure {
		registerDeviceViaMQTT(config)
		// configure should return cleanly if succeeded
		os.Exit(0)
	}

	if *doInstall {
		install(s)
		// install should return cleanly if succeeded
		os.Exit(0)
	}

	go checkAndInstallDependencies()

	err = s.Run()
	check(err, "RunService")
}

func appendIfUnique(slice []string, element string) []string {
	for _, el := range slice {
		if el == element {
			return slice
		}
	}
	slice = append(slice, element)
	return slice
}

func (p program) run() {
	// Export the proxy info as environments variables, so that:
	// - http.DefaultTransport can use the proxy settings
	// - any spawned sketch process'es also have access to them
	// Note, all_proxy will not be used by any HTTP/HTTPS connections.
	p.exportProxyEnvVars()
	p.exportConfigWhitelistedEnvVars()

	// Start nats-server on localhost:4222
	opts := server.Options{}
	opts.Port = 4222
	opts.Host = "127.0.0.1"
	// Remove any host/ip that points to itself in Route
	newroutes, err := server.RemoveSelfReference(opts.Cluster.Port, opts.Routes)
	if err != nil {
		fmt.Println(err)
		return
	}
	opts.Routes = newroutes
	s := server.New(&opts)
	configureNatsdLogger(s, &opts)
	go s.Start()

	if !s.ReadyForConnections(1 * time.Second) {
		log.Fatal("NATS server not redy for connections!")
	}

	// Create global status
	status := NewStatus(p.Config, nil, nil, "$aws/things/"+p.Config.ID)
	status.Update(p.Config)

	// Setup MQTT connection
	certPemPath := filepath.Join(p.Config.CertPath, "certificate.pem")
	certKeyPath := filepath.Join(p.Config.CertPath, "certificate.key")
	mqttClient, err := setupMQTTConnection(certPemPath, certKeyPath, p.Config.ID, p.Config.URL, status)

	if err == nil {
		log.Println("Connected to MQTT")
		status.mqttClient = mqttClient
	} else {
		log.Printf("Connection to MQTT failed, cloud features unavailable: %v", err)

		// TODO: temporary, fail if no connection is available
		os.Exit(0)
	}

	if p.listenFile != "" {
		go tailAndReport(p.listenFile, status)
	}

	// Setup docker daemon connection
	cli, err := docker.NewClientWithOpts(docker.WithVersion("1.38"))

	if err != nil {
		log.Printf("Connection to Docker Daemon failed, containers features unavailable: %v", err)
	}
	status.dockerClient = cli

	// Start nats-client for local server
	nc, err := nats.Connect(nats.DefaultURL)
	check(err, "ConnectNATS")
	_, err = nc.Subscribe("$arduino.cloud.*", natsCloudCB(status))
	if err != nil {
		fmt.Println(err)
		return
	}

	// wipe the thing shadows
	if status.mqttClient != nil {
		mqttClient.Publish("$aws/things/"+p.Config.ID+"/shadow/delete", 1, false, "")
	}

	// start heartbeat
	if status.mqttClient != nil {
		newHeartbeat(func(payload string) error {
			if !status.Info("/heartbeat", payload) {
				return fmt.Errorf("Publish failed")
			}
			return nil
		})
	}

	sketchFolder, err := getSketchFolder(status)
	if err != nil {
		fmt.Println(err)
		return
	}
	// Export LD_LIBRARY_PATH to local lib subfolder
	// This way any external library can be safely copied there and the sketch should run anyway
	os.Setenv("LD_LIBRARY_PATH", filepath.Join(sketchFolder, "lib")+":"+os.Getenv("LD_LIBRARY_PATH"))

	addIntelLibrariesToLdPath()

	files, err := ioutil.ReadDir(sketchFolder)
	if err == nil {
		for _, file := range files {

			//add all files as sketches, stopped, without any PID
			if file.IsDir() {
				continue
			}
			addFileToSketchDB(file, status)
		}
	}

	err = os.Mkdir("/tmp/sketches", 0700)
	if err != nil {
		return
	}

	go addWatcherForManuallyAddedSketches("/tmp/sketches", sketchFolder, status)

	autospawnSketchIfMatchesName("sketchLoadedThroughUSB", status)

	select {}
}

func autospawnSketchIfMatchesName(name string, status *Status) {
	if status.Sketches[name] != nil {
		err := applyAction(status.Sketches[name], "START", status)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func subscribeTopics(mqttClient mqtt.Client, id string, status *Status) {
	// Subscribe to topics endpoint
	if status == nil {
		return
	}
	subscribeTopic(mqttClient, id, "/status/post", status, status.StatusEvent, false)
	subscribeTopic(mqttClient, id, "/upload/post", status, status.UploadEvent, true)
	subscribeTopic(mqttClient, id, "/sketch/post", status, status.SketchEvent, true)
	subscribeTopic(mqttClient, id, "/update/post", status, status.UpdateEvent, true)
	subscribeTopic(mqttClient, id, "/stats/post", status, status.StatsEvent, false)
	subscribeTopic(mqttClient, id, "/wifi/post", status, status.WiFiEvent, true)
	subscribeTopic(mqttClient, id, "/ethernet/post", status, status.EthEvent, true)

	subscribeTopic(mqttClient, id, "/apt/get/post", status, status.AptGetEvent, false)
	subscribeTopic(mqttClient, id, "/apt/list/post", status, status.AptListEvent, false)
	subscribeTopic(mqttClient, id, "/apt/install/post", status, status.AptInstallEvent, true)
	subscribeTopic(mqttClient, id, "/apt/update/post", status, status.AptUpdateEvent, true)
	subscribeTopic(mqttClient, id, "/apt/upgrade/post", status, status.AptUpgradeEvent, true)
	subscribeTopic(mqttClient, id, "/apt/remove/post", status, status.AptRemoveEvent, true)

	subscribeTopic(mqttClient, id, "/apt/repos/list/post", status, status.AptRepositoryListEvent, false)
	subscribeTopic(mqttClient, id, "/apt/repos/add/post", status, status.AptRepositoryAddEvent, true)
	subscribeTopic(mqttClient, id, "/apt/repos/remove/post", status, status.AptRepositoryRemoveEvent, true)
	subscribeTopic(mqttClient, id, "/apt/repos/edit/post", status, status.AptRepositoryEditEvent, true)

	subscribeTopic(mqttClient, id, "/containers/ps/post", status, status.ContainersPsEvent, false)
	subscribeTopic(mqttClient, id, "/containers/images/post", status, status.ContainersListImagesEvent, false)
	subscribeTopic(mqttClient, id, "/containers/action/post", status, status.ContainersActionEvent, true)
	subscribeTopic(mqttClient, id, "/containers/rename/post", status, status.ContainersRenameEvent, true)
}

func subscribeTopic(client mqtt.Client, id, topic string, s *Status, statusHandler mqtt.MessageHandler, isWriteFsRequiredForTopic bool) {
	handler := statusHandler
	if s.config.CheckRoFs && isWriteFsRequiredForTopic {
		handler = func(client mqtt.Client, msg mqtt.Message) {
			mountRootFilesystemRw()
			statusHandler(client, msg)
			mountRootFilesystemRo()
		}
	}

	completeTopic := s.topicPertinence + topic
	if debugMqtt {
		debugHandler := func(client mqtt.Client, msg mqtt.Message) {
			fmt.Println("MQTT IN:", string(msg.Topic()), string(msg.Payload()))
			handler(client, msg)
		}

		if token := client.Subscribe(completeTopic, 1, debugHandler); token.Wait() && token.Error() != nil {
			fmt.Println(token.Error())
		}
	} else {
		if token := client.Subscribe(completeTopic, 1, handler); token.Wait() && token.Error() != nil {
			fmt.Println(token.Error())
		}
	}
}

func isWriteFs() bool {
	f, err := os.OpenFile(".arduino-connector.w", os.O_WRONLY|os.O_CREATE, 0777)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

func mountRootFilesystemRw() {
	if !isWriteFs() {
		rwCmd := exec.Command("mount", "-o", "rw,remount", "/")
		if out, err := rwCmd.CombinedOutput(); err != nil {
			fmt.Println("Failed to remount RW")
			fmt.Println(string(out))
		}
	}
}

func mountRootFilesystemRo() {
	rwCmd := exec.Command("mount", "-o", "ro,remount", "/")
	if out, err := rwCmd.CombinedOutput(); err != nil {
		fmt.Println("Failed to remount RO")
		fmt.Println(string(out))
	}

}

func addFileToSketchDB(file os.FileInfo, status *Status) *SketchStatus {
	id, err := getSketchIDFromDB(file.Name(), status)
	if err != nil {
		id = file.Name()
	}
	fmt.Println("Getting sketch from " + id + " " + file.Name())
	s := SketchStatus{
		ID:     id,
		PID:    0,
		Name:   file.Name(),
		Status: "STOPPED",
	}
	status.Set(id, &s)
	status.Publish()
	return &s
}

func copyFileAndRemoveOriginal(src string, dst string) error {
	// Read all content of src to data
	data, err := ioutil.ReadFile(src)
	if err != nil {
		return err
	}
	// Write data to dst
	err = ioutil.WriteFile(dst, data, 0644)
	if err != nil {
		return err
	}
	os.Remove(src)
	if err != nil {
		return err
	}
	return nil
}

func addWatcherForManuallyAddedSketches(folderOrigin, folderDest string, status *Status) {
	watcher, err := fsnotify.NewWatcher()
	defer func() {
		err = watcher.Close()
		if err != nil {
			panic(err)
		}
	}()
	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				log.Println("event:", event)
				if event.Op&fsnotify.Create == fsnotify.Create {
					// give it some time to settle
					time.Sleep(2 * time.Second)
					//name := filepath.Base(strings.TrimSuffix(event.Name, filepath.Ext(event.Name)))
					//filename := filepath.Join(folderDest, name)
					filename := filepath.Join(folderDest, "sketchLoadedThroughUSB")

					// stop already running sketch if it exists
					if sketch, ok := status.Sketches["sketchLoadedThroughUSB"]; ok {
						err = applyAction(sketch, "STOP", status)
					}

					err = os.Rename(event.Name, filename)
					if err != nil {
						// copy the file and remote the original
						err = copyFileAndRemoveOriginal(event.Name, filename)
						if err != nil {
							// nevermind, break and do nothing
							break
						}
					}
					err = os.Chmod(filename, 0700)
					if err != nil {
						fmt.Println(err)
						return
					}
					log.Println("Moving new sketch to sketches folder")
					fileInfo, errOS := os.Stat(filename)
					if errOS != nil {
						log.Println("Got error:" + err.Error())
						break
					}
					s := addFileToSketchDB(fileInfo, status)
					err = applyAction(s, "START", status)
					if err != nil {
						fmt.Println(err)
						return
					}
				}
			case err = <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()
	err = watcher.Add(folderOrigin)
	if err != nil {
		log.Fatal(err)
	}
	<-done
}

func tailAndReport(listenFile string, status *Status) {
	t, err := tail.TailFile(listenFile, tail.Config{Follow: true})
	for err != nil {
		// retry until the file appears
		time.Sleep(1 * time.Second)
		t, err = tail.TailFile(listenFile, tail.Config{Follow: true})
	}
	for line := range t.Lines {
		if strings.Contains(line.Text, "$$$") {
			status.Info("/install", line.Text)
		}
	}
}

func (p program) exportProxyEnvVars() {
	os.Setenv("http_proxy", p.Config.HTTPProxy)
	os.Setenv("https_proxy", p.Config.HTTPSProxy)
	os.Setenv("all_proxy", p.Config.ALLProxy)

	if os.Getenv("no_proxy") == "" {
		// export the no_proxy env var, if empty
		os.Setenv("no_proxy", "localhost,127.0.0.1,localaddress,.localdomain.com")
	}
}

func (p program) exportConfigWhitelistedEnvVars() {
	for _, envVar := range strings.Split(p.Config.EnvVarsToLoad, ",") {
		if strings.Contains(envVar, "=") {
			envTuple := strings.Split(envVar, "=")
			os.Setenv(envTuple[0], envTuple[1])
		}
	}
}

func check(err error, context string) {
	if err != nil {
		log.Fatal(context, " - ", err)
	}
}

// setupMQTTConnection establish a connection with aws iot
func setupMQTTConnection(cert, key, id, url string, status *Status) (mqtt.Client, error) {
	fmt.Println("setupMQTT", cert, key, id, url)
	// Read certificate
	cer, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, errors.Wrap(err, "read certificate")
	}

	// AutoReconnect option is true by default
	// CleanSession option is true by default
	// KeepAlive option is 30 seconds by default
	opts := mqtt.NewClientOptions() // This line is different, we use the constructor function instead of creating the instance ourselves.
	opts.SetClientID(id)
	opts.SetMaxReconnectInterval(20 * time.Second)
	opts.SetConnectTimeout(30 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetOnConnectHandler(func(c mqtt.Client) {
		subscribeTopics(c, id, status)
	})
	opts.SetTLSConfig(&tls.Config{
		Certificates: []tls.Certificate{cer},
		ServerName:   url,
	})

	port := 8883
	path := "/mqtt"
	brokerURL := fmt.Sprintf("tcps://%s:%d%s", url, port, path)
	opts.AddBroker(brokerURL)

	// mqtt.DEBUG = log.New(os.Stdout, "DEBUG: ", log.Lshortfile)
	mqtt.ERROR = log.New(os.Stdout, "ERROR: ", log.Lshortfile)
	mqtt.WARN = log.New(os.Stdout, "WARN: ", log.Lshortfile)
	mqtt.CRITICAL = log.New(os.Stdout, "CRITICAL: ", log.Lshortfile)

	mqttClient := mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		return nil, errors.Wrap(token.Error(), "connect to mqtt")
	}
	return mqttClient, nil
}

func configureNatsdLogger(s *server.Server, opts *server.Options) {
	var log server.Logger
	colors := true
	// Check to see if stderr is being redirected and if so turn off color
	// Also turn off colors if we're running on Windows where os.Stderr.Stat() returns an invalid handle-error
	stat, err := os.Stderr.Stat()
	if err != nil || (stat.Mode()&os.ModeCharDevice) == 0 {
		colors = false
	}
	log = logger.NewStdLogger(opts.Logtime, opts.Debug, opts.Trace, colors, true)

	s.SetLogger(log, opts.Debug, opts.Trace)
}

// checkAndInstallDependencies wraps all the dependencies installation steps that uses apt and needs to be executed sequentially
func checkAndInstallDependencies() {
	checkAndInstallDocker()
	checkAndInstallNetworkManager()
}
