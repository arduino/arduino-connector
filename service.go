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
	go p.run()
	return nil
}

// Stop doesn nothing
func (p *program) Stop(s service.Service) error {
	return nil
}

func main() {
	workingDirectory, _ := osext.ExecutableFolder()

	svcConfig := &service.Config{
		Name:             "ArduinoConnector",
		DisplayName:      "Arduino Connector Service",
		Description:      "Cloud connector and launcher for Intel IoT devices.",
		WorkingDirectory: workingDirectory,
	}

	prg := &Program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}
	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}
	if service.Interactive() {
		log.Println("Installing service")
		s.Install()
	}
	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}
