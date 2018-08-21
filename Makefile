# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get

.PHONY: all test clean
all: test build
build:
		$(GOBUILD) -ldflags "-X main.version=1.0.0-dev" github.com/arduino/arduino-connector
test:
		$(GOBUILD) -ldflags "-X main.version=1.0.0-dev" github.com/arduino/arduino-connector
		cd ./test && ./upload_dev_artifacts_on_s3.sh
		cd ./test && vagrant up --provision
		cd ./test && ./create_iot_device.sh
		$(GOTEST) -v ./...
		cd ./test && ./teardown_iot_device.sh
		cd ./test && vagrant halt