package main

import (
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/arduino/arduino-connector/updater"
	"github.com/kardianos/osext"
)

func (s *Status) Update(config Config) {

	path, err := osext.Executable()
	if err != nil {
		//c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	var up = &updater.Updater{
		CurrentVersion: version,
		APIURL:         config.updateUrl,
		BinURL:         config.updateUrl,
		DiffURL:        "",
		Dir:            "update/",
		CmdName:        config.appName,
	}

	err = up.BackgroundRun()

	if err != nil {
		return
	}

	//c.JSON(200, gin.H{"success": "Please wait a moment while the agent reboots itself"})
	go restart(path)
}

func restart(path string) {
	log.Println("called restart", path)
	// relaunch ourself and exit
	// the relaunch works because we pass a cmdline in
	// that has serial-port-json-server only initialize 5 seconds later
	// which gives us time to exit and unbind from serial ports and TCP/IP
	// sockets like :8989
	log.Println("Starting new spjs process")

	// figure out current path of executable so we know how to restart
	// this process using osext
	exePath, err3 := osext.Executable()
	if err3 != nil {
		log.Printf("Error getting exe path using osext lib. err: %v\n", err3)
	}

	if path == "" {
		log.Printf("exePath using osext: %v\n", exePath)
	} else {
		exePath = path
	}

	exePath = strings.Trim(exePath, "\n")

	cmd := exec.Command(exePath)

	fmt.Println(cmd)

	err := cmd.Start()
	if err != nil {
		log.Printf("Got err restarting spjs: %v\n", err)
	}
	log.Fatal("Exited current spjs for restart")
}
