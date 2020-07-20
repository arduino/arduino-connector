#!/bin/bash

mosquitto &
sleep 5
go test -v --run="TestDockerPsApi"