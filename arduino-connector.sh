#!/bin/bash -e

# logfile=download.log
# exec > $logfile 2>&1

has() {
	type "$1" > /dev/null 2>&1
	return $?
}

download() {
	if has "wget"; then
		wget -nc $1
	elif has "curl"; then
		curl -SOL $1
	else
		echo "Error: you need curl or wget to proceed" >&2;
		exit 20
	fi
}

# Replicate env variables in uppercase format
export ID=$id
export TOKEN=$token
export HTTP_PROXY=$http_proxy
export HTTPS_PROXY=$https_proxy
export ALL_PROXY=$all_proxy

echo printenv
echo ---------

# cd $HOME
echo home folder
echo ---------

echo remove old files: rm -f arduino-connector* certificate*
echo ---------
rm -f arduino-connector* certificate*

echo download connector
echo ---------
download https://downloads.arduino.cc/tools/arduino-connector

chmod +x arduino-connector

echo $password | sudo -kS -E ./arduino-connector -install > arduino-connector.log 2>&1
echo $password | sudo -kS service ArduinoConnector start