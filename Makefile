# need to use bash in order to use source command
SHELL := /bin/bash

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOTEST_TIMEOUT=20m
GOGET=$(GOCMD) get


.PHONY: all test clean

all: test build

build:
		$(GOBUILD) -ldflags "-X main.version=1.0.0-dev" github.com/arduino/arduino-connector

test: setup-test integ-test teardown-test

setup-test:
		cd ./test && vagrant up --no-provision
		cd ./test && ./create_iot_device.sh

integ-test:
		$(GOBUILD) -ldflags "-X main.version=1.0.0-dev" github.com/arduino/arduino-connector
		cd ./test && ./upload_dev_artifacts_on_s3.sh
		cd ./test && vagrant provision
		source ./test/setup_host_test_env.sh && $(GOTEST) ./... -timeout $(GOTEST_TIMEOUT)

teardown-test:
		cd ./test && ./teardown_iot_device.sh
		cd ./test && vagrant destroy --force
		cd ./test && ./teardown_dev_artifacts.sh
