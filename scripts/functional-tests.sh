#!/usr/bin/env bash

set -euo pipefail

trap 'kill "$(pidof mosquitto)"' EXIT

mosquitto > /dev/null &
go test -race -v --tags=functional --run="TestDocker"
go test -race -v --run="TestUninstall"
go test -race -v --run=TestApt
go test -v ./auth