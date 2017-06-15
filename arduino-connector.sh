#/bin/bash -e

logfile=download.log
exec > $logfile 2>&1

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

cd $HOME
echo home folder
echo ---------
ls
echo remove old files: rm -f arduino-connector* certificate*
echo ----------
rm -f arduino-connector* certificate*
ls
echo move files: mv /tmp/arduino-connector.cfg /tmp/certificate.pem /tmp/certificate.key $HOME
echo ----------
mv /tmp/arduino-connector.cfg /tmp/certificate.pem /tmp/certificate.key $HOME
ls
echo download connector
echo ----------
download https://downloads.arduino.cc/tools/arduino-connector
ls
chmod +x arduino-connector
./arduino-connector > arduino-connector.log 2>&1 &