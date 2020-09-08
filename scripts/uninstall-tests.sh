#!/usr/bin/env bash

set -euo pipefail

trap 'kill "$(pidof mosquitto)"' EXIT

mosquitto > /dev/null &
go test -v --run="TestUninstall"