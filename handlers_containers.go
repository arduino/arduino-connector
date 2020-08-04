//
//  This file is part of arduino-connector
//
//  Copyright (C) 2017-2020  Arduino AG (http://www.arduino.cc/)
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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/arduino/go-apt-client"
	dockerConfig "github.com/docker/cli/cli/config"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	docker "github.com/docker/docker/client"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/pkg/errors"
	"github.com/shirou/gopsutil/host"
	"golang.org/x/net/context"
)

// RunPayload Struct merges connector specific parameters with
// docker API structures ContainerConfig, ContainerHostConfig, NetworkNetworkingConfig
type RunPayload struct {
	ImageName               string                   `json:"image"`
	ContainerName           string                   `json:"name"`
	ContainerID             string                   `json:"id"`
	Action                  string                   `json:"action"`
	User                    string                   `json:"user,omitempty"`
	Password                string                   `json:"password,omitempty"`
	SaveRegistryCredentials bool                     `json:"save_registry_credentials,omitempty"`
	ContainerConfig         container.Config         `json:"container_config,omitempty"`
	ContainerHostConfig     container.HostConfig     `json:"host_config,omitempty"`
	NetworkNetworkingConfig network.NetworkingConfig `json:"networking_config,omitempty"`
}

type PsPayload struct {
	ContainerID string `json:"id,omitempty"`
}

type ImagesPayload struct {
	ImageName string `json:"name,omitempty"`
}

type ChangeNamePayload struct {
	ContainerID   string `json:"id"`
	ContainerName string `json:"name"`
}

// ContainersPsEvent send info about "docker ps -a" command
func (s *Status) ContainersPsEvent(client mqtt.Client, msg mqtt.Message) {
	psPayload := PsPayload{}
	err := json.Unmarshal(msg.Payload(), &psPayload)
	if err != nil {
		s.Error("/containers/ps", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
		return
	}

	containerListOptions := types.ContainerListOptions{All: true}
	if psPayload.ContainerID != "" {
		containerListOptions.Filters = filters.NewArgs(filters.Arg("id", psPayload.ContainerID))
	}

	containers, err := s.dockerClient.ContainerList(context.Background(), containerListOptions)
	if err != nil {
		s.Error("/containers/ps", fmt.Errorf("Json marshal result: %s", err))
		return
	}

	data, err := json.Marshal(containers)
	if err != nil {
		s.Error("/containers/ps", fmt.Errorf("Json marsahl result: %s", err))
		return
	}

	s.SendInfo(s.topicPertinence+"/containers/ps", string(data)+"\n")
}

// ContainersListImagesEvent implements docker images
func (s *Status) ContainersListImagesEvent(client mqtt.Client, msg mqtt.Message) {
	imagesPayload := ImagesPayload{}
	err := json.Unmarshal(msg.Payload(), &imagesPayload)
	if err != nil {
		s.Error("/containers/images", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
		return
	}

	imageListOptions := types.ImageListOptions{All: true}
	if imagesPayload.ImageName != "" {
		imageListOptions.Filters = filters.NewArgs(filters.Arg("reference", imagesPayload.ImageName))
	}

	images, err := s.dockerClient.ImageList(context.Background(), imageListOptions)
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

	s.SendInfo(s.topicPertinence+"/containers/images", string(data)+"\n")
}

// ContainersRenameEvent implements docker rename
func (s *Status) ContainersRenameEvent(client mqtt.Client, msg mqtt.Message) {
	cnPayload := ChangeNamePayload{}
	err := json.Unmarshal(msg.Payload(), &cnPayload)
	if err != nil {
		s.Error("/containers/rename", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
		return
	}
	err = s.dockerClient.ContainerRename(context.Background(), cnPayload.ContainerID, cnPayload.ContainerName)
	if err != nil {
		s.Error("/containers/rename", fmt.Errorf("rename result: %s", err))
		return
	}

	// Send result
	data, err := json.Marshal(cnPayload)
	if err != nil {
		s.Error("/containers/rename", fmt.Errorf("Json marsahl result: %s", err))
		return
	}

	s.SendInfo(s.topicPertinence+"/containers/rename", string(data)+"\n")
}

// ContainersActionEvent implements docker container action like run, start and stop, remove
func (s *Status) ContainersActionEvent(client mqtt.Client, msg mqtt.Message) {

	runParams := RunPayload{}
	err := json.Unmarshal(msg.Payload(), &runParams)
	if err != nil {
		s.Error("/containers/action", errors.Wrapf(err, "unmarshal %s", msg.Payload()))
		return
	}

	runResponse := RunPayload{
		ImageName:     runParams.ImageName,
		ContainerName: runParams.ContainerName,
		ContainerID:   runParams.ContainerID,
		Action:        runParams.Action,
	}

	ctx := context.Background()
	switch runParams.Action {
	case "run":
		// check if user and passw are present in order to add auth
		// remember that the imageName should provide also the registry endpoint
		// i.e 6435543362.dkr.ecr.eu-east-1.amazonaws.com/redis:latest
		// the default is  docker.io/library/redis:latest
		pullOpts, authConfig, errConf := ConfigureRegistryAuth(runParams)
		if errConf != nil {
			fmt.Println(errConf)
			break
		}

		if authConfig != nil {
			_, err = s.dockerClient.RegistryLogin(ctx, *authConfig)
			if err != nil {
				ClearRegistryAuth(runParams)
				s.Error("/containers/action", fmt.Errorf("auth test failed: %s", err))
				return
			}
		}
		out, errPull := s.dockerClient.ImagePull(ctx, runParams.ImageName, pullOpts)
		if errPull != nil {
			s.Error("/containers/action", fmt.Errorf("image pull result: %s", errPull))
			return
		}
		// waiting the complete download of the image
		_, err = io.Copy(ioutil.Discard, out)
		if err != nil {
			fmt.Println(err)
		}

		defer out.Close()
		fmt.Fprintf(os.Stdout, "Successfully downloaded image: %s\n", runParams.ImageName)

		// overwrite imagename in container.Config
		runParams.ContainerConfig.Image = runParams.ImageName
		// by default bind all the exposed ports via PublishAllPorts if the field PortBindings is empty
		// note that in this case docker decide the host port an the port changes if the container is restarted
		if len(runParams.ContainerHostConfig.PortBindings) == 0 {
			runParams.ContainerHostConfig.PublishAllPorts = true
		}

		resp, errCreate := s.dockerClient.ContainerCreate(ctx, &runParams.ContainerConfig, &runParams.ContainerHostConfig,
			&runParams.NetworkNetworkingConfig, runParams.ContainerName)

		if errCreate != nil {
			s.Error("/containers/action", fmt.Errorf("container create result: %s", errCreate))
			return
		}

		if err = s.dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
			s.Error("/containers/action", fmt.Errorf("container start result: %s", err))
			return
		}
		runResponse.ContainerID = resp.ID
		fmt.Fprintf(os.Stdout, "Successfully started container %s from  Image: %s\n", resp.ID, runParams.ImageName)

	case "stop":
		if err = s.dockerClient.ContainerStop(ctx, runParams.ContainerID, nil); err != nil {
			s.Error("/containers/action", fmt.Errorf("container action result: %s", err))
			return
		}
		fmt.Fprintf(os.Stdout, "Successfully stopped container %s\n", runParams.ContainerID)

	case "start":
		if err = s.dockerClient.ContainerStart(ctx, runParams.ContainerID, types.ContainerStartOptions{}); err != nil {
			s.Error("/containers/action", fmt.Errorf("container action result: %s", err))
			return
		}
		fmt.Fprintf(os.Stdout, "Successfully started container %s\n", runParams.ContainerID)

	case "remove":
		forceAllOption := types.ContainerRemoveOptions{
			Force:         true,
			RemoveLinks:   false,
			RemoveVolumes: true,
		}

		if err = s.dockerClient.ContainerRemove(ctx, runParams.ContainerID, forceAllOption); err != nil {
			s.Error("/containers/action", fmt.Errorf("container remove result: %s", err))
			return
		}
		fmt.Fprintf(os.Stdout, "Successfully removed container %s\n", runParams.ContainerID)
		// implements docker image prune -a that removes all images not associated to a container
		forceAllImagesArg, _ := filters.FromJSON(`{"dangling": false}`)
		if _, errPrune := s.dockerClient.ImagesPrune(ctx, forceAllImagesArg); err != nil {
			s.Error("/containers/action", fmt.Errorf("images prune result: %s", errPrune))
			return
		}
		fmt.Fprintf(os.Stdout, "Successfully pruned container images\n")

	default:
		s.Error("/containers/action", fmt.Errorf("container command %s not found", runParams.Action))
		return
	}

	// Send result
	data, err := json.Marshal(runResponse)
	if err != nil {
		s.Error("/containers/action", fmt.Errorf("Json marshal result: %s", err))
		return
	}

	s.Info("/containers/action", string(data)+"\n")

}

// ConfigureRegistryAuth manages registry authentication usage flow
func ConfigureRegistryAuth(runParams RunPayload) (types.ImagePullOptions, *types.AuthConfig, error) {
	var authConfig *types.AuthConfig
	var err error
	pullOpts := types.ImagePullOptions{}
	imageRegistryEndpoint := strings.Split(runParams.ImageName, "/")[0]
	if runParams.User != "" && runParams.Password != "" {
		// user provided credentials
		authConfig = &types.AuthConfig{
			Username:      runParams.User,
			Password:      runParams.Password,
			ServerAddress: imageRegistryEndpoint,
		}
		encodedJSON, errMars := json.Marshal(authConfig)
		if errMars != nil {
			return pullOpts, authConfig, errMars
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)
		pullOpts = types.ImagePullOptions{RegistryAuth: authStr}
		// if user requested save, do the save of the credentials in ~/.docker/config.json in the docker standard way
		if runParams.SaveRegistryCredentials {
			loadedConfigFile, errLoad := dockerConfig.Load(dockerConfig.Dir())
			if errLoad != nil {
				return pullOpts, authConfig, errLoad
			}
			loadedConfigFile.AuthConfigs[authConfig.ServerAddress] = *authConfig
			err = loadedConfigFile.Save()
			if err != nil {
				fmt.Println(err)
			}
		}

	} else {
		//user did not provided credentials, search in config
		loadedConfigFile, errLoad := dockerConfig.Load(dockerConfig.Dir())
		if errLoad != nil {
			return pullOpts, authConfig, err
		}
		loadedAuthConfig, errConf := loadedConfigFile.GetAuthConfig(imageRegistryEndpoint)
		if errConf != nil {
			return pullOpts, authConfig, err
		}
		authConfig = &loadedAuthConfig
		encodedJSON, errMar := json.Marshal(loadedAuthConfig)
		if errMar != nil {
			return pullOpts, authConfig, err
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)
		pullOpts = types.ImagePullOptions{RegistryAuth: authStr}

	}
	return pullOpts, authConfig, err
}

// ClearRegistryAuth removes credential for a certain registry from docker config
func ClearRegistryAuth(runParams RunPayload) {
	loadedConfigFile, err := dockerConfig.Load(dockerConfig.Dir())
	if err != nil {
		return
	}
	imageRegistryEndpoint := strings.Split(runParams.ImageName, "/")[0]
	delete(loadedConfigFile.AuthConfigs, imageRegistryEndpoint)
	err = loadedConfigFile.Save()
	if err != nil {
		fmt.Println(err)
	}
}

// checkAndInstallDocker implements steps from https://docs.docker.com/install/linux/docker-ce/ubuntu/
func checkAndInstallDocker() {
	cli, err := docker.NewClientWithOpts(docker.WithVersion("1.38"))
	defer func() {
		err = cli.Close()
		if err != nil {
			return
		}
	}()
	if cli != nil {
		_, err = cli.ContainerList(context.Background(), types.ContainerListOptions{})
		if err != nil {
			fmt.Println("Docker daemon not found!")
		}
	}
	if err != nil {
		//returns  platform string, family string, version string, err error
		platform, family, version, err := host.PlatformInformation()
		distroVer, cerr := strconv.Atoi(strings.Replace(version, ".", "", -1))
		if err != nil && cerr != nil {
			fmt.Println("Failed to fetch system info")
		}
		fmt.Printf("Fetched system info: %s %s %s on arch: %s\n", platform, family, version, runtime.GOARCH)
		if runtime.GOARCH == "amd64" {
			if platform == "ubuntu" {
				if distroVer >= 1604 {
					installDockerCEOnXenialAndNewer()
				}
			}
		} else if runtime.GOARCH == "arm" {
			if platform == "raspbian" {
				installDockerCEViaConvenienceScript()
			}
		}
	}
}

func installDockerCEViaConvenienceScript() {
	curlString := "curl -fsSL get.docker.com -o get-docker.sh"
	curlCmd := exec.Command("bash", "-c", curlString)
	if out, errCmd := curlCmd.CombinedOutput(); errCmd != nil {
		fmt.Println("Failed to Download Docker CE Convenience Script Installer:")
		fmt.Println(string(out))
	}

	installCmd := exec.Command("sh", "get-docker.sh")
	if out, err := installCmd.CombinedOutput(); err != nil {
		fmt.Println("Failed to Run Docker CE Convenience Script Installer:")
		fmt.Println(string(out))
	}
}

func installDockerCEOnXenialAndNewer() {
	// dpkg --configure -a for prevent block of installation
	dpkgCmd := exec.Command("dpkg", "--configure", "-a")
	if out, err := dpkgCmd.CombinedOutput(); err != nil {
		fmt.Println("Failed to reconfigure dpkg:")
		fmt.Println(string(out))
	}

	_, err := apt.CheckForUpdates()
	if err != nil {
		fmt.Println(err)
		return
	}

	dockerPrerequisitesPackages := []*apt.Package{
		&apt.Package{Name: "apt-transport-https"},
		&apt.Package{Name: "ca-certificates"},
		&apt.Package{Name: "curl"},
		&apt.Package{Name: "software-properties-common"},
	}
	for _, pac := range dockerPrerequisitesPackages {
		if out, errInstall := apt.Install(pac); errInstall != nil {
			fmt.Println("Failed to install: ", pac.Name)
			fmt.Println(string(out))
			return
		}
	}

	curlString := "curl -fsSL https://download.docker.com/linux/ubuntu/gpg | apt-key add -"
	curlCmd := exec.Command("bash", "-c", curlString)
	if out, errCmd := curlCmd.CombinedOutput(); errCmd != nil {
		fmt.Println("Failed to add Docker’s official GPG key:")
		fmt.Println(string(out))
	}

	repoString := `add-apt-repository "deb [arch=amd64] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable"`
	repoCmd := exec.Command("bash", "-c", repoString)
	if out, errCmd := repoCmd.CombinedOutput(); errCmd != nil {
		fmt.Println("Failed to set up the stable docker repository:")
		fmt.Println(string(out))
	}

	_, err = apt.CheckForUpdates()
	if err != nil {
		fmt.Println(err)
		return
	}

	toInstall := &apt.Package{Name: "docker-ce"}
	if out, err := apt.Install(toInstall); err != nil {
		fmt.Println("Failed to install docker-ce:")
		fmt.Println(string(out))
		return
	}

	sysCmd := exec.Command("systemctl", "enable", "docker")
	if out, err := sysCmd.CombinedOutput(); err != nil {
		fmt.Println("Failed to systemctl enable docker:")
		fmt.Println(string(out))
	}
}
