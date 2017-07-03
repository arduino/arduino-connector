// simple does nothing except block while running the service.
package main

import (
	"log"

	"github.com/kardianos/osext"
	"github.com/kardianos/service"
)

var logger service.Logger

type program struct{}

// Start run the program asynchronously
func (p *program) Start(s service.Service) error {
	// go p.run()
	return nil
}

// Stop doesn nothing
func (p *program) Stop(s service.Service) error {
	return nil
}

// createService returns the servcie to be installed
func createService() service.Service {
	workingDirectory, _ := osext.ExecutableFolder()

	svcConfig := &service.Config{
		Name:             "ArduinoConnector",
		DisplayName:      "Arduino Connector Service",
		Description:      "Cloud connector and launcher for Intel IoT devices.",
		WorkingDirectory: workingDirectory,
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	check(err)

	return s
}

func setup(token string) {}

// install creates a devices with the arduino api, along with a key and certificate, and it installs itself as a service
func install(s service.Service) {
	log.Println("Install as a service")
	err := s.Install()
	check(err)
}
