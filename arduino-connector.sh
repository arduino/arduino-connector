#/bin/bash -e

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

cd ~
mv /tmp/arduino-connector.cfg /tmp/certificate.pem /tmp/certificate.key ~
rm -f arduino-connector
download https://downloads.arduino.cc/tools/arduino-connector
chmod +x arduino-connector
./arduino-connector > arduino-connector.log 2>&1 &