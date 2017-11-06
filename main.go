//
//  This file is part of arduino-connector
//
//  Copyright (C) 2017  Arduino AG (http://www.arduino.cc/)
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
	"path/filepath"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/fsnotify/fsnotify"
	"github.com/hpcloud/tail"
	"github.com/namsral/flag"
	logger "github.com/nats-io/gnatsd/logger"
	server "github.com/nats-io/gnatsd/server"
	nats "github.com/nats-io/go-nats"

	"github.com/pkg/errors"
)

const (
	configFile = "./arduino-connector.cfg"
)

// Config holds the configuration needed by the application
type Config struct {
	ID         string
	URL        string
	HTTPProxy  string
	HTTPSProxy string
	ALLProxy   string
}

func (c Config) String() string {
	out := "id=" + c.ID + "\r\n"
	out += "url=" + c.URL + "\r\n"
	out += "http_proxy=" + c.HTTPProxy + "\r\n"
	out += "https_proxy=" + c.HTTPSProxy + "\r\n"
	out += "all_proxy=" + c.ALLProxy + "\r\n"
	return out
}

func main() {
	config := Config{}

	var doLogin = flag.Bool("login", false, "Do the login and prints out a temporary token")
	var doInstall = flag.Bool("install", false, "Install as a service")
	var doRegister = flag.Bool("register", false, "Registers on the cloud")
	var listenFile = flag.String("listen", "", "Tail given file and report percentage")
	var token = flag.String("token", "", "an authentication token")
	flag.String(flag.DefaultConfigFlagname, "", "path to config file")
	flag.StringVar(&config.ID, "id", "", "id of the thing in aws iot")
	flag.StringVar(&config.URL, "url", "", "url of the thing in aws iot")
	flag.StringVar(&config.HTTPProxy, "http_proxy", "", "URL of HTTP proxy to use")
	flag.StringVar(&config.HTTPSProxy, "https_proxy", "", "URL of HTTPS proxy to use")
	flag.StringVar(&config.ALLProxy, "all_proxy", "", "URL of SOCKS proxy to use")

	flag.Parse()

	if *doLogin {
		token, err := askCredentials()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		fmt.Println("Access Token:", token)
		os.Exit(0)
	}

	if *doRegister {
		register(config, *token)
	}

	// Create service and install
	s, err := createService(config, *listenFile)
	check(err, "CreateService")

	if *doInstall {
		install(s)
		// install should return cleanly if succeeded
		os.Exit(0)
	}

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

	// Start nats-server on localhost:4222
	opts := server.Options{}
	opts.Port = 4222
	opts.Host = "127.0.0.1"
	// Remove any host/ip that points to itself in Route
	newroutes, err := server.RemoveSelfReference(opts.Cluster.Port, opts.Routes)
	opts.Routes = newroutes
	s := server.New(&opts)
	configureNatsdLogger(s, &opts)
	go s.Start()

	if !s.ReadyForConnections(1 * time.Second) {
		log.Fatal("NATS server not redy for connections!")
	}

	// Create global status
	status := NewStatus(p.Config.ID, nil)

	// Setup MQTT connection
	mqttClient, err := setupMQTTConnection("certificate.pem", "certificate.key", p.Config.ID, p.Config.URL, status)

	if err == nil {
		log.Println("Connected to MQTT")
		status.mqttClient = mqttClient
	} else {
		log.Println("Connection to MQTT failed, cloud features unavailable")
		// TODO: temporary, fail if no connection is available
		os.Exit(0)
	}

	if p.listenFile != "" {
		go tailAndReport(p.listenFile, status)
	}

	// Start nats-client for local server
	nc, err := nats.Connect(nats.DefaultURL)
	check(err, "ConnectNATS")
	nc.Subscribe("$arduino.cloud.*", NatsCloudCB(status))

	// wipe the thing shadows
	if status.mqttClient != nil {
		mqttClient.Publish("$aws/things/"+p.Config.ID+"/shadow/delete", 1, false, "")
	}

	sketchFolder, err := GetSketchFolder()
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

	os.Mkdir("/tmp/sketches", 0777)

	go addWatcherForManuallyAddedSketches("/tmp/sketches", sketchFolder, status)

	autospawnSketchIfMatchesName("sketchLoadedThroughUSB", status)

	select {}
}

func autospawnSketchIfMatchesName(name string, status *Status) {
	if status.Sketches[name] != nil {
		applyAction(status.Sketches[name], "START", status)
	}
}

func subscribeTopics(mqttClient mqtt.Client, id string, status *Status) {
	// Subscribe to topics endpoint
	if status == nil {
		return
	}
	mqttClient.Subscribe("$aws/things/"+id+"/status/post", 1, StatusCB(status))
	mqttClient.Subscribe("$aws/things/"+id+"/upload/post", 1, UploadCB(status))
	mqttClient.Subscribe("$aws/things/"+id+"/sketch/post", 1, SketchCB(status))
	mqttClient.Subscribe("$aws/things/"+id+"/update/post", 1, UpdateCB(status))
}

func addFileToSketchDB(file os.FileInfo, status *Status) *SketchStatus {
	id, err := GetSketchIDFromDB(file.Name())
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
	defer watcher.Close()
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

					err := os.Rename(event.Name, filename)
					if err != nil {
						// copy the file and remote the original
						err = copyFileAndRemoveOriginal(event.Name, filename)
						if err != nil {
							// nevermind, break and do nothing
							break
						}
					}
					os.Chmod(filename, 0755)
					log.Println("Moving new sketch to sketches folder")
					fileInfo, err := os.Stat(filename)
					if err != nil {
						log.Println("Got error:" + err.Error())
						break
					}
					s := addFileToSketchDB(fileInfo, status)
					applyAction(s, "START", status)
				}
			case err := <-watcher.Errors:
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
	opts.SetConnectTimeout(0)
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
