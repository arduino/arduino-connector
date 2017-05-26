package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/kardianos/osext"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/vharitonsky/iniflags"
)

func main() {
	var (
		id  = flag.String("id", "", "id of the thing in aws iot")
		url = flag.String("url", "", "url of the thing in aws iot")
	)

	// Read configuration
	iniflags.SetConfigFile("./arduino_connector.cfg")
	iniflags.Parse()

	// Setup MQTT connection
	client, err := setupMQTTConnection("cert.pem", "private.key", *id, *url)
	check(err)
	log.Println("Connected to MQTT")

	// Get uuid
	uid, _, err := getUUID()
	check(err)

	// Register
	// if created {
	err = registerDevice(*id, uid, client)
	check(err)
	log.Println("Registered device")
	// }

	// Create global status
	status := NewStatus(*id, client)

	// Subscribe to /upload endpoint
	client.Subscribe("$aws/things/"+*id+"/upload", 1, UploadCB(status))
	client.Subscribe("$aws/things/"+*id+"/sketch", 1, SketchCB(status))

	select {}
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

// UploadPayload contains the name and url of the sketch to upload on the device
type UploadPayload struct {
	URL  string
	Name string
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
			status.Error(errors.Wrapf(err, "unmarshal %s", msg.Payload()))
			return
		}

		// create folder
		folder, err := osext.ExecutableFolder()
		if err != nil {
			status.Error(errors.Wrapf(err, "create sketch folder %s", msg.Payload()))
			return
		}

		// download the binary
		name := filepath.Join(folder, info.Name)
		err = downloadFile(name, info.URL)
		if err != nil {
			status.Error(errors.Wrapf(err, "download file %s", info.URL))
			return
		}

		// chmod it
		err = os.Chmod(name, 0744)
		if err != nil {
			status.Error(errors.Wrapf(err, "chmod 744 %s", name))
			return
		}

		// spawn process
		pid, stdout, err := spawnProcess(name)
		if err != nil {
			status.Error(errors.Wrapf(err, "spawn %s", name))
			return
		}

		status.Info("Sketch started with PID " + strconv.Itoa(pid))

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
			status.Error(errors.Wrapf(err, "unmarshal %s", msg.Payload()))
			return
		}

		if sketch, ok := status.Sketches[payload.Name]; ok {
			err = applyAction(sketch, payload.Action)
			if err != nil {
				status.Error(errors.Wrapf(err, "applying %s to %s", payload.Action, payload.Name))
				return
			}
		}

		status.Error(errors.New("sketch " + payload.Name + " not found"))
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
	opts.SetTLSConfig(&tls.Config{Certificates: []tls.Certificate{cer}})

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
		log.Println(token.Error())
		return nil, errors.Wrap(token.Error(), "connect to mqtt")
	}
	return mqttClient, nil
}

// getUUID get a previously created uuid or creates a new one, and returns it along with a flag that says if it's been created
// Can fail if the file is corrupted or there are missing permissions
func getUUID() (uuid.UUID, bool, error) {
	var id uuid.UUID

	idFile, err := ioutil.ReadFile("uuid")
	if err != nil {
		id = uuid.NewV4()
		err := ioutil.WriteFile("uuid", []byte(id.String()), 0600)
		if err != nil {
			return id, true, errors.Wrap(err, "save the uuid on file")
		}
		return id, true, nil
	}

	id, err = uuid.FromString(string(idFile))
	if err != nil {
		return id, false, errors.Wrap(err, "decode uuid file")
	}
	return id, false, nil
}

// registerDevice publishes on the topic /register with info about the device itself
func registerDevice(id string, uid uuid.UUID, client mqtt.Client) error {
	// get host
	host, err := os.Hostname()
	if err != nil {
		return errors.Wrap(err, "get hostname")
	}

	// get Macs
	macs, err := getMACs()

	data := struct {
		ID   uuid.UUID
		Host string
		MACs []string
	}{
		ID:   uid,
		Host: host,
		MACs: macs,
	}
	msg, err := json.Marshal(data)
	if err != nil {
		return errors.Wrapf(err, "marshal %+v to json", data)
	}

	if token := client.Publish("$aws/things/"+id+"/register", 1, false, msg); token.Wait() && token.Error() != nil {
		return errors.Wrap(token.Error(), "publish to /register")
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

// downloadfile substitute a file with something that downloads from an url
func downloadFile(filepath string, url string) error {
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
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
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
