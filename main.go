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
	"github.com/hpcloud/tail"
	"github.com/namsral/flag"
	logger "github.com/nats-io/gnatsd/logger"
	server "github.com/nats-io/gnatsd/server"
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

	// Create service and install
	s, err := createService(config, *listenFile)
	check(err, "CreateService")

	if *doRegister {
		register(config, *token)
	}

	if *doInstall {
		install(s)
	}

	err = s.Run()
	check(err, "RunService")
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
	configureLogger(s, &opts)
	go s.Start()

	// Setup MQTT connection
	client, err := setupMQTTConnection("certificate.pem", "certificate.key", p.Config.ID, p.Config.URL)
	if err != nil {
		// if installing in a chroot the paths may be wrong and the installer may fail.
		// Don't report it as an error
		os.Exit(0)
	}
	log.Println("Connected to MQTT")

	// Create global status
	status := NewStatus(p.Config.ID, client)

	if p.listenFile != "" {
		go tailAndReport(p.listenFile, status)
	}

	// Subscribe to topics endpoint
	client.Subscribe("$aws/things/"+p.Config.ID+"/status/post", 1, StatusCB(status))
	client.Subscribe("$aws/things/"+p.Config.ID+"/upload/post", 1, UploadCB(status))
	client.Subscribe("$aws/things/"+p.Config.ID+"/sketch/post", 1, SketchCB(status))

	sketchFolder, err := GetSketchFolder()
	// Export LD_LIBRARY_PATH to local lib subfolder
	// This way any external library can be safely copied there and the sketch should run anyway
	os.Setenv("LD_LIBRARY_PATH", filepath.Join(sketchFolder, "lib")+":$LD_LIBRARY_PATH")

	files, err := ioutil.ReadDir(sketchFolder)

	if err == nil {
		for _, file := range files {

			//add all files as sketches, stopped, without any PID
			if file.IsDir() {
				continue
			}
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
		}
	}

	select {}
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
func setupMQTTConnection(cert, key, id, url string) (mqtt.Client, error) {
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
	opts.SetMaxReconnectInterval(1 * time.Second)
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

func configureLogger(s *server.Server, opts *server.Options) {
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
