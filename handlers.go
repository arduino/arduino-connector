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
	"golang.org/x/crypto/ssh/terminal"
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

type DylibMap struct {
	Name     string   `json:"Name"`
	Provides []string `json:"Provides"`
	URL      string   `json:"URL"`
	Help     string   `json:"Help"`
}

func (d *DylibMap) Download(path string) {
	for _, element := range d.Provides {
		resp, err := http.Get(d.URL + "/" + element)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		filePath := filepath.Join(path, element)
		ioutil.WriteFile(filePath, body, 0600)
	}
}

func (d *DylibMap) Contains(match string) bool {
	for _, element := range d.Provides {
		if strings.Contains(element, match) {
			return true
		}
	}
	return false
}

func downloadDylibDependencies(library string) error {
	resp, err := http.Get("https://downloads.arduino.cc/libArduino/dylib_dependencies.txt")
	if err != nil {
		return errors.New("Can't download dylibs registry")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 { // OK
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.New("Can't read dylibs registry")
		}
		var v []DylibMap
		err = json.Unmarshal(bodyBytes, &v)
		if err != nil {
			return err
		}
		for _, element := range v {
			if element.Contains(library) {
				folder, _ := GetSketchFolder()
				fmt.Println(element.Help)
				if element.Help != "" {
					// TODO: remove and replace with a status.Info()
					return errors.New(element.Help)
				}
				element.Download(filepath.Join(folder, "lib"))
			}
		}
		return errors.New("Can't find a provider for " + library)
	}
	return nil
}

func extractLibrary(errorString string) string {
	fields := strings.Fields(errorString)
	for _, subStr := range fields {
		if strings.Contains(subStr, ".so") {
			subStr = strings.TrimRight(subStr, ":")
			libName := strings.Split(subStr, ".")
			if len(libName) >= 2 {
				return libName[0] + "." + libName[1]
			}
		}
	}
	return ""
}

func checkForLibrariesMissingError(filepath string, sketch *SketchStatus, status *Status, err string) {
	if strings.Contains(err, "error while loading shared libraries") {
		// download dependencies and retry
		// if the error persists, bail out
		addIntelLibrariesToLdPath()
		fmt.Println("Missing library!")
		library := extractLibrary(err)
		status.Info("/upload", "Downloading needed libraries")
		err_download := downloadDylibDependencies(library)
		if err_download != nil {
			status.Error("/upload", err_download)
		}
		status.Error("/upload", errors.New("Missing libraries, install them and relaunch the sketch"))
	}
}

func checkSketchForMissingDisplayEnvVariable(errorString string, filepath string, sketch *SketchStatus, status *Status) {
	if strings.Contains(errorString, "Can't open display") || strings.Contains(errorString, "cannot open display") {

		if os.Getenv("DISPLAY") == "NULL" {
			os.Setenv("DISPLAY", ":0")
			return
		}

		err := setupDisplay(true)
		if err != nil {
			setupDisplay(false)
		}
		spawnProcess(filepath, sketch, status)
		sketch.Status = "RUNNING"
	}
}

func setupDisplay(usermode bool) error {
	// Blindly set DISPLAY env variable to default
	i := 0
	for {
		os.Setenv("DISPLAY", ":"+strconv.Itoa(i))
		fmt.Println("Exporting DISPLAY as " + ":" + strconv.Itoa(i))
		// Unlock xorg session for localhost connections
		// TODO: find a way to automatically remove -nolisten tcp
		cmd := exec.Command("xhost", "+localhost")
		if usermode {
			cmd.SysProcAttr = &syscall.SysProcAttr{}
			cmd.SysProcAttr.Credential = &syscall.Credential{Uid: 1000, Gid: 1000}
		}
		out, err := cmd.CombinedOutput()
		fmt.Println(string(out))
		// Also try xrandr
		cmd = exec.Command("xrandr")
		out, err_xrandr := cmd.CombinedOutput()
		fmt.Println(string(out))
		if err != nil || err_xrandr != nil {
			if i > 2 {
				fmt.Println("Xorg server unavailable, make sure you have a display attached and a user logged in")
				fmt.Println("If it's already ok, try setting up Xorg to accept incoming connection (-listen tcp)")
				fmt.Println("On Ubuntu, add \n\n[SeatDefaults]\nxserver-allow-tcp=true\n\nto /etc/lightdm/lightdm.conf")
				os.Setenv("DISPLAY", "NULL")
				return errors.New("Unable to open display")
			}
		} else {
			return nil
		}
		i++
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

	terminal.MakeRaw(int(f.Fd()))

	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderr_buf.String())
		return 0, stdout, stderr, err
	}

	sketch.pty = f
	if status.mqttClient != nil {
		go status.mqttClient.Subscribe("$aws/things/"+status.id+"/stdin", 1, StdInCB(f, status))
	}

	go func() {
		for {
			temp := make([]byte, 1000)
			len, err := f.Read(temp)
			if err != nil {
				break
			}
			if len > 0 {
				fmt.Println(string(temp[:len]))
				status.Raw("/stdout", string(temp[:len]))
				checkForLibrariesMissingError(filepath, sketch, status, string(temp))
				checkSketchForMissingDisplayEnvVariable(string(temp), filepath, sketch, status)
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
		if sketch.PID != 0 && err == nil && process.Pid != 0 {
			fmt.Println("kill called")
			err = process.Kill()
		} else {
			err = nil
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
