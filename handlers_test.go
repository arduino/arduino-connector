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
	"context"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// ExecAsVagrantSshCmd "wraps vagrant ssh -c
func ExecAsVagrantSshCmd(command string) (string, error) {
	vagrantSSHCmd := fmt.Sprintf(`cd test && vagrant ssh -c "%s"`, command)
	cmd := exec.Command("bash", "-c", vagrantSSHCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// tests
func TestSketchProcessIsRunning(t *testing.T) {
	mqtt := NewMqttTestClient()
	defer mqtt.Close()
	sketchTopic := "upload"

	fs := http.FileServer(http.Dir("test/sketch_devops_integ_test"))
	http.DefaultServeMux = new(http.ServeMux)
	http.Handle("/", fs)

	srv := &http.Server{Addr: ":3000"}

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			fmt.Println(err)
		}
	}()

	sketchDownloadCommand := fmt.Sprintf(`{"token": "","url": "%s","name": "sketch_devops_integ_test.elf","id": "0774e17e-f60e-4562-b87d-18017b6ef3d2"}`, "http://10.0.2.2:3000/sketch_devops_integ_test.elf")
	responseSketchRun := mqtt.MqttSendAndReceiveSync(t, sketchTopic, sketchDownloadCommand)
	t.Log(responseSketchRun)

	assert.Equal(t, true, strings.Contains(responseSketchRun, "INFO: Sketch started with PID "))
	pid := strings.TrimSuffix(strings.Split(responseSketchRun, "INFO: Sketch started with PID ")[1], "\n")
	outputMessage, err := ExecAsVagrantSshCmd(fmt.Sprintf("ps -p %s --no-headers", pid))
	t.Log(outputMessage)

	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, 1, len(strings.Split(strings.TrimSuffix(outputMessage, "\n"), "\n")))
	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()
	if err := srv.Shutdown(ctx); err != nil {
		t.Error(err)
	}
}

// tests
func TestMaliciousSketchProcessIsNotRunning(t *testing.T) {
	mqtt := NewMqttTestClient()
	defer mqtt.Close()
	sketchTopic := "upload"

	fs := http.FileServer(http.Dir("test/sketch_devops_integ_test/sketch_devops_integ_test_malicious"))
	http.DefaultServeMux = new(http.ServeMux)
	http.Handle("/", fs)
	srv := &http.Server{Addr: ":3000"}

	go func() {
		err := srv.ListenAndServe()
		if err != nil {
			fmt.Println(err)
		}
	}()

	sketchDownloadCommand := fmt.Sprintf(`{"token": "","url": "%s","name": "sketch_devops_integ_test.elf","id": "0774e17e-f60e-4562-b87d-18017b6ef3d2"}`, "http://10.0.2.2:3000/sketch_devops_integ_test.elf")
	responseSketchRun := mqtt.MqttSendAndReceiveSync(t, sketchTopic, sketchDownloadCommand)
	t.Log(responseSketchRun)

	assert.Equal(t, true, strings.Contains(responseSketchRun, "ERROR: signature do not match"))
	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()
	if err := srv.Shutdown(ctx); err != nil {
		t.Error(err)
	}
}

func TestSketchProcessHasConfigWhitelistedEnvVars(t *testing.T) {
	// see upload_dev_artifacts_on_s3.sh to see where env vars are passed to the config
	mqtt := NewMqttTestClient()
	defer mqtt.Close()

	//test connector config
	outputMessage, err := ExecAsVagrantSshCmd("sudo cat /root/arduino-connector.cfg")
	if err != nil {
		t.Error(err)
	}
	envString := outputMessage
	t.Log(envString)
	assert.Equal(t, true, strings.Contains(envString, "env_vars_to_load=HDDL_INSTALL_DIR=/opt/intel/computer_vision_sdk/inference_engine/external/hddl/,ENV_TEST_PATH=/tmp"))

	//test environment
	sketchTopic := "upload"

	fs := http.FileServer(http.Dir("test/sketch_env_integ_test"))
	http.DefaultServeMux = new(http.ServeMux)
	http.Handle("/", fs)

	srv := &http.Server{Addr: ":3000"}

	go func() {
		err = srv.ListenAndServe()
		if err != nil {
			fmt.Println(err)
		}
	}()

	sketchDownloadCommand := fmt.Sprintf(`{"token": "","url": "%s","name": "connector_env_var_test.bin","id": "0774e17e-f60e-4562-b87d-18017b6ef3d2"}`, "http://10.0.2.2:3000/connector_env_var_test.bin")
	responseSketchRun := mqtt.MqttSendAndReceiveSync(t, sketchTopic, sketchDownloadCommand)
	t.Log(responseSketchRun)

	assert.Equal(t, true, strings.Contains(responseSketchRun, "INFO: Sketch started with PID "))

	outputMessage, err = ExecAsVagrantSshCmd("cat /tmp/printenv.out")
	if err != nil {
		t.Error(err)
	}

	envString = outputMessage
	t.Log(envString)

	assert.Equal(t, true, strings.Contains(envString, "HDDL_INSTALL_DIR=/opt/intel/computer_vision_sdk/inference_engine/external/hddl/"))
	assert.Equal(t, true, strings.Contains(envString, "ENV_TEST_PATH=/tmp"))
	ctx, cancelFunc := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancelFunc()
	if err := srv.Shutdown(ctx); err != nil {
		t.Error(err)
	}

}
