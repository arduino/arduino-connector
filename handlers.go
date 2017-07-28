package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/kardianos/osext"
	"github.com/kr/pty"
	nats "github.com/nats-io/go-nats"
	"github.com/pkg/errors"
)

// StatusCB replies with the current status of the arduino-connector
func StatusCB(status *Status) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		status.Publish()
	}
}

// UploadPayload contains the name and url of the sketch to upload on the device
type UploadPayload struct {
	ID    string `json:"id"`
	URL   string `json:"url"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

// UploadCB receives the url and name of the sketch binary, then it
// - downloads the binary,
// - chmods +x it
// - executes redirecting stdout and sterr to a proper logger
func UploadCB(status *Status) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		// unmarshal
		var info UploadPayload
		var sketch SketchStatus
		err := json.Unmarshal(msg.Payload(), &info)
		if err != nil {
			status.Error("/upload", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
			return
		}

		if info.ID == "" {
			info.ID = info.Name
		}

		// Stop and delete if existing
		if sketch, ok := status.Sketches[info.ID]; ok {
			err = applyAction(sketch, "STOP", status)
			if err != nil {
				status.Error("/upload", errors.Wrapf(err, "stop pid %d", sketch.PID))
				return
			}

			sketchFolder, err := GetSketchFolder()
			err = os.Remove(filepath.Join(sketchFolder, sketch.Name))
			if err != nil {
				status.Error("/upload", errors.Wrapf(err, "remove %d", sketch.Name))
				return
			}
		}

		folder, err := GetSketchFolder()
		if err != nil {
			status.Error("/upload", errors.Wrapf(err, "create sketch folder %s", info.ID))
			return
		}

		// download the binary
		name := filepath.Join(folder, info.Name)
		err = downloadFile(name, info.URL, info.Token)
		if err != nil {
			status.Error("/upload", errors.Wrapf(err, "download file %s", info.URL))
			return
		}

		// chmod it
		err = os.Chmod(name, 0744)
		if err != nil {
			status.Error("/upload", errors.Wrapf(err, "chmod 744 %s", name))
			return
		}

		sketch.ID = info.ID
		sketch.Name = info.Name
		// save ID-Name to a sort of DB
		InsertSketchInDB(sketch.Name, sketch.ID)

		// spawn process
		pid, _, _, err := spawnProcess(name, &sketch, status)
		if err != nil {
			status.Error("/upload", errors.Wrapf(err, "spawn %s", name))
			return
		}

		status.Info("/upload", "Sketch started with PID "+strconv.Itoa(pid))

		sketch.PID = pid
		sketch.Status = "RUNNING"

		status.Set(info.ID, &sketch)
		status.Publish()

		// go func(stdout io.ReadCloser) {
		// 	in := bufio.NewScanner(stdout)
		// 	for {
		// 		for in.Scan() {
		// 			fmt.Printf(in.Text()) // write each line to your log, or anything you need
		// 		}
		// 	}
		// }(stdout)
	}
}

func GetSketchFolder() (string, error) {
	// create folder if it doesn't exist
	folder, err := osext.ExecutableFolder()
	folder = filepath.Join(folder, "sketches")
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		err = os.Mkdir(folder, 0644)
	}
	return folder, err
}

func GetSketchDBFolder() (string, error) {
	// create folder if it doesn't exist
	folder, err := GetSketchFolder()
	folder = filepath.Join(folder, "db")
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		err = os.Mkdir(folder, 0644)
	}
	return folder, err
}

func GetSketchDB() (string, error) {
	// create folder if it doesn't exist
	folder, err := GetSketchDBFolder()
	if err != nil {
		return "", err
	}
	db := filepath.Join(folder, "db")
	return db, err
}

func InsertSketchInDB(name string, id string) {
	// create folder if it doesn't exist
	db, err := GetSketchDB()
	if err != nil {
		return
	}

	var c []SketchBinding
	raw, err := ioutil.ReadFile(db)
	json.Unmarshal(raw, &c)

	for _, element := range c {
		if element.ID == id && element.Name == name {
			return
		}
	}
	c = append(c, SketchBinding{ID: id, Name: name})
	data, _ := json.Marshal(c)
	ioutil.WriteFile(db, data, 0755)
}

func GetSketchIDFromDB(name string) (string, error) {
	// create folder if it doesn't exist
	db, err := GetSketchDB()
	if err != nil {
		return "", errors.New("Can't open DB")
	}
	var c []SketchBinding
	raw, err := ioutil.ReadFile(db)
	json.Unmarshal(raw, &c)

	for _, element := range c {
		if element.Name == name {
			return element.ID, nil
		}
	}
	return "", errors.New("No matching sketch")
}

// SketchActionPayload contains the name of the sketch and the action to perform
type SketchActionPayload struct {
	ID     string
	Name   string
	Action string
}

// SketchCB listens to commands to start and stop sketches
func SketchCB(status *Status) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		// unmarshal
		var info SketchActionPayload
		err := json.Unmarshal(msg.Payload(), &info)
		if err != nil {
			status.Error("/sketch", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
			return
		}

		if info.ID == "" {
			info.ID = info.Name
		}

		if sketch, ok := status.Sketches[info.ID]; ok {
			err := applyAction(sketch, info.Action, status)
			if err != nil {
				status.Error("/sketch", errors.Wrapf(err, "applying %s to %s", info.Action, info.Name))
				return
			}
			status.Info("/sketch", "successfully performed "+info.Action+" on sketch "+info.ID)

			status.Set(info.ID, sketch)
			status.Publish()
			return
		}

		status.Error("/sketch", errors.New("sketch "+info.ID+" not found"))
	}
}

func NatsCloudCB(s *Status) nats.MsgHandler {
	return func(m *nats.Msg) {
		thingName := strings.TrimPrefix(m.Subject, "$arduino.cloud.")

		updateMessage := fmt.Sprintf("{\"state\": {\"reported\": { \"%s\": %s}}}", thingName, string(m.Data))

		s.mqttClient.Publish("$aws/things/"+s.id+"/shadow/update", 1, false, updateMessage)
	}
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

func logSketchStdoutStderr(cmd *exec.Cmd, stdout io.ReadCloser, stderr io.ReadCloser, sketch *SketchStatus) {
	stdoutCopy := bufio.NewScanner(stdout)
	stderrCopy := bufio.NewScanner(stderr)

	stdoutCopy.Split(bufio.ScanLines)
	stderrCopy.Split(bufio.ScanLines)

	go func() {
		fmt.Println("started scanning stdout")
		for stdoutCopy.Scan() {
			fmt.Printf(stdoutCopy.Text())
		}
	}()

	go func() {
		fmt.Println("started scanning stderr")
		for stderrCopy.Scan() {
			fmt.Printf(stderrCopy.Text())
		}
	}()
}

func StdInCB(pty *os.File, status *Status) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		if len(msg.Payload()) > 0 {
			pty.Write(msg.Payload())
		}
	}
}

// spawn Process creates a new process from a file
func spawnProcess(filepath string, sketch *SketchStatus, status *Status) (int, io.ReadCloser, io.ReadCloser, error) {
	cmd := exec.Command(filepath)
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	var stderr_buf bytes.Buffer
	cmd.Stderr = &stderr_buf

	f, err := pty.Start(cmd)

	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr_buf.String())
		return 0, stdout, stderr, err
	}

	sketch.pty = f
	//status.client.Subscribe("$aws/things/"+status.id+"/"+strconv.Itoa(cmd.Process.Pid)+"/stdin", 1, StdInCB(f, status))

	go func() {
		for {
			temp := make([]byte, 1000)
			len, err := f.Read(temp)
			if err != nil {
				break
			}
			if len > 0 {
				fmt.Println(string(temp))
				status.Info("/stdout", string(temp))
			}
		}
	}()

	//logSketchStdoutStderr(cmd, stdout, stderr, sketch)

	// keep track of sketch life (and isgnal if it ends abruptly)
	go func() {
		err := cmd.Wait()
		//if we get here signal that the sketch has died
		applyAction(sketch, "STOP", status)
		if err != nil {
			fmt.Println(fmt.Sprint(err) + ": " + stderr_buf.String())
		}
		fmt.Println("sketch exited " + err.Error())
	}()

	return cmd.Process.Pid, stdout, stderr, err
}

func applyAction(sketch *SketchStatus, action string, status *Status) error {
	process, err := os.FindProcess(sketch.PID)
	if err != nil && sketch.PID != 0 {
		fmt.Println("exit because of error")
		return err
	}

	switch action {
	case "START":
		if sketch.PID != 0 {
			err = process.Signal(syscall.SIGCONT)
		} else {
			folder, err := GetSketchFolder()
			if err != nil {
				return err
			}
			name := filepath.Join(folder, sketch.Name)
			sketch.PID, _, _, err = spawnProcess(name, sketch, status)
		}
		if err != nil {
			return err
		}
		sketch.Status = "RUNNING"
		break

	case "STOP":
		fmt.Println("stop called")
		if sketch.PID != 0 && err == nil {
			fmt.Println("kill called")
			err = process.Kill()
		}
		sketch.PID = 0
		sketch.Status = "STOPPED"
		break
	case "PAUSE":
		err = process.Signal(syscall.SIGTSTP)
		sketch.Status = "PAUSED"
		break
	}
	return err
}
