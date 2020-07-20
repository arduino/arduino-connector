#!/usr/bin/env bash

set -euo pipefail

trap 'kill "$(pidof mosquitto)"' EXIT

mosquitto &
sleep 5
go test -v --run="TestDockerPsApi"