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
	"encoding/json"
	"fmt"
	"os/exec"

	apt "github.com/arduino/go-apt-client"
	"github.com/docker/docker/api/types"
	mqtt "github.com/eclipse/paho.mqtt.golang"

	docker "github.com/docker/docker/client"
	"golang.org/x/net/context"
)

func checkAndInstallDocker() {
	// fmt.Println("try to install docker-ce")
	cli, err := docker.NewEnvClient()
	if err != nil {
		fmt.Println("Docker daemon not found!")
		fmt.Println(err.Error())
	}
	_, err = cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		fmt.Println("Docker daemon not found!")
		fmt.Println(err.Error())
	}

	if err != nil {
		go func() {
			//steps from https://docs.docker.com/install/linux/docker-ce/ubuntu/
			apt.CheckForUpdates()
			dockerPrerequisitesPackages := []*apt.Package{&apt.Package{Name: "apt-transport-https"}, &apt.Package{Name: "ca-certificates"}, &apt.Package{Name: "curl"}, &apt.Package{Name: "software-properties-common"}}
			for _, pac := range dockerPrerequisitesPackages {
				if out, err := apt.Install(pac); err != nil {
					fmt.Println("Failed to install: ", pac.Name)
					fmt.Println(string(out))
					return
				}
			}
			cmdString := "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -"
			cmd := exec.Command("bash", "-c", cmdString)
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Println("Failed to add Docker’s official GPG key:")
				fmt.Println(string(out))
			}

			repoString := "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"
			cmd = exec.Command("add-apt-repository", repoString)
			if out, err := cmd.CombinedOutput(); err != nil {
				fmt.Println("Failed to set up the stable repository:")
				fmt.Println(string(out))
			}

			apt.CheckForUpdates()
			toInstall := &apt.Package{Name: "docker-ce"}
			if out, err := apt.Install(toInstall); err != nil {
				fmt.Println("Failed to install docker-ce:")
				fmt.Println(string(out))
				return
			}
			// fmt.Println("done to install docker-ce")
		}()
	}

}

// ContainersPsEvent implements docker ps
func (s *Status) ContainersPsEvent(client mqtt.Client, msg mqtt.Message) {
	cli, err := docker.NewEnvClient()
	if err != nil {
		s.Error("/containers/ps", fmt.Errorf("Json marsahl result: %s", err))
		return
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		s.Error("/containers/ps", fmt.Errorf("Json marsahl result: %s", err))
		return
	}

	// Send result
	data, err := json.Marshal(containers)
	if err != nil {
		s.Error("/containers/ps", fmt.Errorf("Json marsahl result: %s", err))
		return
	}

	//var out bytes.Buffer
	//json.Indent(&out, data, "", "  ")
	//fmt.Println(string(out.Bytes()))

	s.Info("/containers/ps", string(data)+"\n")
}

// ContainersListImagesEvent implements docker images
func (s *Status) ContainersListImagesEvent(client mqtt.Client, msg mqtt.Message) {
	// Gather images
	cli, err := docker.NewEnvClient()
	if err != nil {
		s.Error("/containers/images", fmt.Errorf("images result: %s", err))
		return
	}

	images, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	if err != nil {
		s.Error("/containers/images", fmt.Errorf("images result: %s", err))
		return
	}

	// Send result
	data, err := json.Marshal(images)
	if err != nil {
		s.Error("/containers/images", fmt.Errorf("Json marsahl result: %s", err))
		return
	}

	//var out bytes.Buffer
	//json.Indent(&out, data, "", "  ")
	//fmt.Println(string(out.Bytes()))

	s.Info("/containers/images", string(data)+"\n")
}
