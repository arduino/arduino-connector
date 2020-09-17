#!/bin/bash -e

#
#  This file is part of arduino-connector
#
#  Copyright (C) 2017-2020  Arduino AG (http://www.arduino.cc/)
#
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.
#

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

cd $HOME
echo home folder
echo ---------

echo remove old files
echo ---------
rm -f /usr/bin/arduino-connector* /usr/bin/certificate*

echo uninstall previous installations of connector
echo ---------
if [ "$password" == "" ]
then
	sudo systemctl stop ArduinoConnector || true
else
	echo $password | sudo -kS systemctl stop ArduinoConnector || true
fi

if [ "$password" == "" ]
then
	sudo rm -f /etc/systemd/system/ArduinoConnector.service
else
	echo $password | sudo -kS rm -f /etc/systemd/system/ArduinoConnector.service
fi

echo download connector
echo ---------
download https://downloads.arduino.cc/tools/feed/arduino-connector/arduino-connector-arm
sudo mv arduino-connector-arm /usr/bin/arduino-connector
sudo chmod +x /usr/bin/arduino-connector

echo install connector
echo ---------
if [ "$password" == "" ]
then
	sudo -E arduino-connector -register -install
else
	echo $password | sudo -kS -E arduino-connector -register -install > arduino-connector.log 2>&1
fi

echo start connector service
echo ---------
if [ "$password" == "" ]
then
	sudo systemctl start ArduinoConnector
else
	echo $password | sudo -kS systemctl start ArduinoConnector
fi
