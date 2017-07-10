package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/davecgh/go-spew/spew"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/hpcloud/tail"
	"github.com/kardianos/osext"
	"github.com/namsral/flag"
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

	// Setup MQTT connection
	client, err := setupMQTTConnection("certificate.pem", "certificate.key", p.Config.ID, p.Config.URL)
	check(err, "ConnectMQTT")
	log.Println("Connected to MQTT")

	// Create global status
	status := NewStatus(p.Config.ID, client)

	if p.listenFile != "" {
		go tailAndReport(p.listenFile, status)
	}

	// Subscribe to /upload endpoint
	client.Subscribe("$aws/things/"+p.Config.ID+"/upload/post", 1, UploadCB(status))
	client.Subscribe("$aws/things/"+p.Config.ID+"/sketch", 1, SketchCB(status))

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

// UploadPayload contains the name and url of the sketch to upload on the device
type UploadPayload struct {
	URL   string `json:"url"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

// SketchActionPayload contains the name of the sketch and the action to perform
type SketchActionPayload struct {
	Name   string
	Action string
}

// UploadCB receives the url and name of the sketch binary, then it
// - downloads the binary,
// - chmods +x it
// - executes redirecting stdout and sterr to a proper logger
func UploadCB(status *Status) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		// unmarshal
		var info UploadPayload
		err := json.Unmarshal(msg.Payload(), &info)
		if err != nil {
			status.Error("/upload/get", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
			return
		}

		spew.Dump(msg.Payload())
		spew.Dump(info)

		// create folder
		folder, err := osext.ExecutableFolder()
		if err != nil {
			status.Error("/upload/get", errors.Wrapf(err, "create sketch folder %s", msg.Payload()))
			return
		}

		// download the binary
		name := filepath.Join(folder, info.Name)
		err = downloadFile(name, info.URL, info.Token)
		if err != nil {
			status.Error("/upload/get", errors.Wrapf(err, "download file %s", info.URL))
			return
		}

		// chmod it
		err = os.Chmod(name, 0744)
		if err != nil {
			status.Error("/upload/get", errors.Wrapf(err, "chmod 744 %s", name))
			return
		}

		// spawn process
		pid, stdout, err := spawnProcess(name)
		if err != nil {
			status.Error("/upload/get", errors.Wrapf(err, "spawn %s", name))
			return
		}

		status.Info("/upload/get", "Sketch started with PID "+strconv.Itoa(pid))

		s := SketchStatus{
			PID:    pid,
			Name:   info.Name,
			Status: "RUNNING",
		}
		status.Set(info.Name, s)

		go func(stdout io.ReadCloser) {
			in := bufio.NewScanner(stdout)
			for {
				for in.Scan() {
					fmt.Printf(in.Text()) // write each line to your log, or anything you need
				}
			}
		}(stdout)
	}
}

// SketchCB listens to commands to start and stop sketches
func SketchCB(status *Status) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		// unmarshal
		var payload SketchActionPayload
		err := json.Unmarshal(msg.Payload(), &payload)
		if err != nil {
			status.Error("/sketch/get", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
			return
		}

		if sketch, ok := status.Sketches[payload.Name]; ok {
			err = applyAction(sketch, payload.Action)
			if err != nil {
				status.Error("/sketch/get", errors.Wrapf(err, "applying %s to %s", payload.Action, payload.Name))
				return
			}
		}

		status.Error("/sketch/get", errors.New("sketch "+payload.Name+" not found"))
	}
}

func applyAction(sketch SketchStatus, action string) error {
	process, err := os.FindProcess(sketch.PID)
	if err != nil {
		return err
	}
	switch action {
	case "START":
		err = process.Signal(syscall.SIGCONT)
		sketch.Status = "RUNNING"
		break
	case "STOP":
		err = process.Kill()
		sketch.Status = "STOPPED"
		break
	case "PAUSE":
		err = process.Signal(syscall.SIGTSTP)
		sketch.Status = "PAUSED"
		break
	}
	return err
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

// downloadfile substitute a file with something that downloads from an url
func downloadFile(filepath, url, token string) error {
	// Create the file - remove the existing one if it exists
	if _, err := os.Stat(filepath); err == nil {
		err := os.Remove(filepath)
		if err != nil {
			return errors.Wrap(err, "remove "+filepath)
		}
	}
	out, err := os.Create(filepath)
	if err != nil {
		return errors.Wrap(err, "create "+filepath)
	}
	defer out.Close()
	// Get the data
	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("Expected OK, got " + resp.Status)
	}

	// Writer the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

// spawn Process creates a new process from a file
func spawnProcess(filepath string) (int, io.ReadCloser, error) {
	cmd := exec.Command(filepath)
	stdout, err := cmd.StdoutPipe()
	if err := cmd.Start(); err != nil {
		return 0, stdout, err
	}
	return cmd.Process.Pid, stdout, err
}
