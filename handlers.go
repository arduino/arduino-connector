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
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/kardianos/osext"
	"github.com/kr/pty"
	nats "github.com/nats-io/go-nats"
	"github.com/pkg/errors"
	"golang.org/x/crypto/ssh/terminal"
)

// StatusEvent replies with the current status of the arduino-connector
func (status *Status) StatusEvent(client mqtt.Client, msg mqtt.Message) {
	status.Publish()
}

// UpdateEvent handles the connector autoupdate
// Any URL must be signed with Arduino private key
func (status *Status) UpdateEvent(client mqtt.Client, msg mqtt.Message) {
	var info struct {
		URL       string `json:"url"`
		Signature string `json:"signature"`
		Token     string `json:"token"`
	}
	err := json.Unmarshal(msg.Payload(), &info)
	if err != nil {
		status.Error("/update", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
		return
	}
	executablePath, _ := os.Executable()
	name := filepath.Join(os.TempDir(), filepath.Base(executablePath))
	err = downloadFile(name, info.URL, info.Token)
	err = downloadFile(name+".sig", info.URL+".sig", info.Token)
	if err != nil {
		status.Error("/update", errors.Wrap(err, "no signature file "+info.URL+".sig"))
		return
	}
	// check the signature
	err = checkGPGSig(name, name+".sig")
	if err != nil {
		status.Error("/update", errors.Wrap(err, "wrong signature "+info.URL+".sig"))
		return
	}
	// chmod it
	err = os.Chmod(name, 0755)
	if err != nil {
		status.Error("/update", errors.Wrapf(err, "chmod 755 %s", name))
		return
	}
	os.Rename(executablePath, executablePath+".old")
	// copy it over existing binary
	err = copyFileAndRemoveOriginal(name, executablePath)
	if err != nil {
		// rollback
		os.Rename(executablePath+".old", executablePath)
		status.Error("/update", errors.Wrap(err, "error copying itself from "+name+" to "+executablePath))
		return
	}
	os.Chmod(executablePath, 0755)
	os.Remove(executablePath + ".old")
	// leap of faith: kill itself, systemd should respawn the process
	os.Exit(0)
}

// UploadEvent receives the url and name of the sketch binary, then it
// - downloads the binary,
// - chmods +x it
// - executes redirecting stdout and sterr to a proper logger
func (status *Status) UploadEvent(client mqtt.Client, msg mqtt.Message) {
	var info struct {
		ID    string `json:"id"`
		URL   string `json:"url"`
		Name  string `json:"name"`
		Token string `json:"token"`
	}
	err := json.Unmarshal(msg.Payload(), &info)
	if err != nil {
		status.Error("/upload", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
		return
	}

	if info.ID == "" {
		info.ID = info.Name
	}

	// Stop and delete if existing
	var sketch SketchStatus
	if sketch, ok := status.Sketches[info.ID]; ok {
		err = applyAction(sketch, "STOP", status)
		if err != nil {
			status.Error("/upload", errors.Wrapf(err, "stop pid %d", sketch.PID))
			return
		}

		sketchFolder, err := getSketchFolder()
		err = os.Remove(filepath.Join(sketchFolder, sketch.Name))
		if err != nil {
			status.Error("/upload", errors.Wrapf(err, "remove %d", sketch.Name))
			return
		}
	}

	folder, err := getSketchFolder()
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
	err = os.Chmod(name, 0700)
	if err != nil {
		status.Error("/upload", errors.Wrapf(err, "chmod 700 %s", name))
		return
	}

	sketch.ID = info.ID
	sketch.Name = info.Name
	// save ID-Name to a sort of DB
	insertSketchInDB(sketch.Name, sketch.ID)

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

func getSketchFolder() (string, error) {
	// create folder if it doesn't exist
	folder, err := osext.ExecutableFolder()
	folder = filepath.Join(folder, "sketches")
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		err = os.Mkdir(folder, 0700)
	}
	return folder, err
}

func getSketchDBFolder() (string, error) {
	// create folder if it doesn't exist
	folder, err := getSketchFolder()
	folder = filepath.Join(folder, "db")
	if _, err := os.Stat(folder); os.IsNotExist(err) {
		err = os.Mkdir(folder, 0700)
	}
	return folder, err
}

func getSketchDB() (string, error) {
	// create folder if it doesn't exist
	folder, err := getSketchDBFolder()
	if err != nil {
		return "", err
	}
	db := filepath.Join(folder, "db")
	return db, err
}

func insertSketchInDB(name string, id string) {
	// create folder if it doesn't exist
	db, err := getSketchDB()
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
	ioutil.WriteFile(db, data, 0600)
}

func getSketchIDFromDB(name string) (string, error) {
	// create folder if it doesn't exist
	db, err := getSketchDB()
	if err != nil {
		return "", errors.New("can't open DB")
	}
	var c []SketchBinding
	raw, err := ioutil.ReadFile(db)
	json.Unmarshal(raw, &c)

	for _, element := range c {
		if element.Name == name {
			return element.ID, nil
		}
	}
	return "", errors.New("no matching sketch")
}

// SketchEvent listens to commands to start and stop sketches
func (status *Status) SketchEvent(client mqtt.Client, msg mqtt.Message) {
	var info struct {
		ID     string
		Name   string
		Action string
	}
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

func natsCloudCB(s *Status) nats.MsgHandler {
	return func(m *nats.Msg) {
		thingName := strings.TrimPrefix(m.Subject, "$arduino.cloud.")

		updateMessage := fmt.Sprintf("{\"state\": {\"reported\": { \"%s\": %s}}}", thingName, string(m.Data))

		if s.messagesSent > 1000 {
			fmt.Println("rate limiting: " + strconv.Itoa(s.messagesSent))
			introducedDelay := time.Duration(s.messagesSent/1000) * time.Second
			if introducedDelay > 20*time.Second {
				introducedDelay = 20 * time.Second
			}
			time.Sleep(introducedDelay)
		}
		s.messagesSent++
		s.mqttClient.Publish("$aws/things/"+s.id+"/shadow/update", 1, false, updateMessage)
		if debugMqtt {
			fmt.Println("MQTT OUT: $aws/things/"+s.id+"/shadow/update", updateMessage)
		}
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

func stdInCB(pty *os.File, status *Status) mqtt.MessageHandler {
	return func(client mqtt.Client, msg mqtt.Message) {
		if len(msg.Payload()) > 0 {
			pty.Write(msg.Payload())
		}
	}
}

type dylibMap struct {
	Name     string   `json:"Name"`
	Provides []string `json:"Provides"`
	URL      string   `json:"URL"`
	Help     string   `json:"Help"`
}

func (d *dylibMap) Download(path string) {
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

func (d *dylibMap) Contains(match string) bool {
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
		return errors.New("can't download dylibs registry")
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 { // OK
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.New("can't read dylibs registry")
		}
		var v []dylibMap
		err = json.Unmarshal(bodyBytes, &v)
		if err != nil {
			return err
		}
		for _, element := range v {
			if element.Contains(library) {
				folder, _ := getSketchFolder()
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
		if err := downloadDylibDependencies(library); err != nil {
			status.Error("/upload", err)
		}
		status.Error("/upload", errors.New("missing libraries, install them and relaunch the sketch"))
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
		out, errXhost := cmd.CombinedOutput()
		fmt.Println(string(out))
		// Also try xrandr
		cmd = exec.Command("xrandr")
		out, errXrandr := cmd.CombinedOutput()
		fmt.Println(string(out))
		if errXhost != nil || errXrandr != nil {
			if i > 2 {
				fmt.Println("Xorg server unavailable, make sure you have a display attached and a user logged in")
				fmt.Println("If it's already ok, try setting up Xorg to accept incoming connection (-listen tcp)")
				fmt.Println("On Ubuntu, add \n\n[SeatDefaults]\nxserver-allow-tcp=true\n\nto /etc/lightdm/lightdm.conf")
				os.Setenv("DISPLAY", "NULL")
				return errors.New("unable to open display")
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
	var stderrBuf bytes.Buffer
	cmd.Stderr = &stderrBuf

	f, err := pty.Start(cmd)

	terminal.MakeRaw(int(f.Fd()))

	if err != nil {
		fmt.Println(fmt.Sprint(err) + ": " + stderrBuf.String())
		return 0, stdout, stderr, err
	}

	sketch.pty = f
	if status.mqttClient != nil {
		go status.mqttClient.Subscribe("$aws/things/"+status.id+"/stdin", 1, stdInCB(f, status))
	}

	go func() {
		for {
			temp := make([]byte, 1000)
			len, err := f.Read(temp)
			if err != nil {
				break
			}
			if len > 0 {
				//fmt.Println(string(temp[:len]))
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
			fmt.Println(fmt.Sprint(err) + ": " + stderrBuf.String())
		}
		fmt.Println("sketch exited ")
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
			folder, err := getSketchFolder()
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
	case "DELETE":
		applyAction(sketch, "STOP", status)
		fmt.Println("delete called")
		sketchFolder, err := getSketchFolder()
		err = os.Remove(filepath.Join(sketchFolder, sketch.Name))
		if err != nil {
			fmt.Println("error deleting sketch")
		}
		status.Sketches[sketch.ID] = nil
		break
	case "PAUSE":
		err = process.Signal(syscall.SIGTSTP)
		sketch.Status = "PAUSED"
		break
	}
	return err
}
