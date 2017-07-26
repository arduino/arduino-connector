#!/bin/bash -e

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
rm -f arduino-connector* certificate*

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
download https://downloads.arduino.cc/tools/arduino-connector
chmod +x arduino-connector

echo install connector
echo ---------
if [ "$password" == "" ]
then
	sudo -E ./arduino-connector -install
else
	echo $password | sudo -kS -E ./arduino-connector -register -install > arduino-connector.log 2>&1
fi

if [ "$password" == "" ]
then
	sudo chown $USER arduino-connector.cfg
else
	echo $password | sudo -kS chown $USER arduino-connector.cfg
fi


echo start connector service
echo ---------
if [ "$password" == "" ]
then
	sudo systemctl start ArduinoConnector
else
	echo $password | sudo -kS systemctl start ArduinoConnector
fi
