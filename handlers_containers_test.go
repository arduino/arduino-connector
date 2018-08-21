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
	"os/exec"
	"strings"
	"testing"
	"fmt"
)

func TestConnectorProcessIsRunning(t *testing.T) {
	vagrantCmd:="systemctl status ArduinoConnector | grep running"
	vagrantSSHCmd := fmt.Sprintf(`cd test && vagrant ssh -c "%s"`,vagrantCmd)
	cmd := exec.Command("bash", "-c", vagrantSSHCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Error(err)
	}
	outputMessage:=string(out)
	if !strings.Contains(outputMessage, "active (running)") {
		t.Error(outputMessage)
	}
}

func TestConnectorDockerIsRunning(t *testing.T) {
	vagrantCmd:="sudo docker version"
	vagrantSSHCmd := fmt.Sprintf(`cd test && vagrant ssh -c "%s"`,vagrantCmd)
	cmd := exec.Command("bash", "-c", vagrantSSHCmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Error(err)
	}
	outputMessage:=string(out)
	if !strings.Contains(outputMessage, "Version:") {
		t.Error(outputMessage)
	}
}


