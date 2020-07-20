#!/usr/bin/env bash

set -euo pipefail

trap 'kill "$(pidof mosquitto)"' EXIT

mosquitto &
go test -v --run="TestDockerPsApi"