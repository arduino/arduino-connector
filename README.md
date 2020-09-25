# Arduino Connector

[![Go Report Card](https://goreportcard.com/badge/github.com/arduino/arduino-connector)](https://goreportcard.com/report/github.com/arduino/arduino-connector)
![Build](https://github.com/arduino/arduino-connector/workflows/Build/badge.svg)
![lint](https://github.com/arduino/arduino-connector/workflows/lint/badge.svg)
![Tests](https://github.com/arduino/arduino-connector/workflows/Tests/badge.svg)

The Arduino Connector allows your device to connect to the Arduino Cloud, and push and receive messages through the [MQTT protocol](http://mqtt.org/). You can see and control all your cloud-enabled devices via a web app called [My Devices](https://create.arduino.cc/devices).

## How does it work?

The Arduino Connector gets installed on a device and does the following things:

- Connects to MQTT using the certificate and key generated during installation
- Starts and Stops sketches according to the received commands from MQTT
- Collects the output of the sketches in order to send them on MQTT

## Install

Follow the ["Getting Started"](https://create.arduino.cc/getting-started/) guides to install the connector and allow your devices to communicate with the cloud via Arduino Create. You can install the connector onto a [Up2 board](https://create.arduino.cc/getting-started/up2) or a generic [Intel-based platform running Linux](https://create.arduino.cc/getting-started/intel-platforms).

Make sure you have an Arduino Account and are able to [log in](https://auth.arduino.cc/login).

Please write us at auth@arduino.cc if you encounter any issue logging in and you need support.

## Development for Intel-based platform

- Download Vagrant on you pc (https://www.vagrantup.com/)
- Clone this repository and `cd <folder>`
- `vagrant init debian/jessie64`
- `vagrant plugin install vagrant-vbguest`
- `vagrant up`
- `vagrant ssh`
- `sudo apt update`
- `sudo apt upgrade`
- `sudo apt-get autoremove`
- `exit`
- `vagrant halt`
- Add inside a Vagrantfile the following line `config.vm.synced_folder "./", "/vagrant_data"`
- `vagrant up`

Ok now we have a vagrant machine debian based where install and develop arduino-connector.
Inside the machine follow the getting-started guide from previus link and you should be able to see on your dashboard the data of vagrant machine.

### Develop workflow

- Probably arduino-connector will be installed in `home/vagrant` folder.
- Check status of service with `sudo systemctl status ArduinoConnector.service`
- Stop it `sudo systemctl stop ArduinoConnector.service`
- From host machine build new binary in project folder (`go build -ldflags "-X main.version=$VERSION"`)
- Move binary from `/vagrant_data` (shared folder of repository and vagrant machine) to `/home/vagrant`
- Start service `sudo systemctl start ArduinoConnector.service`
- Check your changes (show logs with ``)

## Build for ARM devices

```bash
GOOS=linux GOARCH=arm go build -ldflags "-X main.version=arm-dev" -o=arduino-connector-arm github.com/arduino/arduino-connector
```

## Autoupdate
```
go get github.com/sanbornm/go-selfupdate
./bin/go-selfupdate arduino-connector $VERSION
# scp -r public/* user@server:/var/www/files/arduino-connector
```

## API documentation

See [API](./API.md)

## Functional tests

These tests can be executed locally. To do that, you need to configure a dedicated docker container:
- get the image `docker pull guerra1994/go-docker-mqtt-ubuntu-env`
- enter the container `docker run -it -v $(pwd):/home --privileged --name gmd guerra1994/go-docker-mqtt-ubuntu-env`
- run the mosquitto MQTT broker in background mode `mosquitto > /dev/null 2>&1 &`
- then run your test, for example `go test -v --tags=functional --run="TestDockerPsApi"`


## Integration tests disclaimer

You will see in the following paragraphs that the testing environment and procedures are strictly coupled with the
Arduino web services. We're sorry of this behaviour because is not so "community friendly" but we are aiming to improve 
both the quality of the connector code and its testing process. Obviously no code quality improvement is possible without
the safety net that tests provide :). So please be patient while we improve the whole process.

## Generate temporary installer script
```
aws-google-auth -p arduino
go build -ldflags "-X main.version=2.0.22" github.com/arduino/arduino-connector
aws --profile arduino s3 cp arduino-connector-dev.sh s3://arduino-tmp/arduino-connector.sh
aws s3 presign --profile arduino s3://arduino-tmp/arduino-connector.sh --expires-in $(expr 3600 \* 24)
#use this link i the wget of the getting started script
aws --profile arduino s3 cp arduino-connector s3://arduino-tmp/
aws s3 presign --profile arduino s3://arduino-tmp/arduino-connector  --expires-in $(expr 3600 \* 24)
# use the output as the argument of arduino-connector-dev.sh qhen launching getting started script:

export id=containtel:a4ae70c4-b7ff-40c8-83c1-1e10ee166241
wget -O install.sh <aws signed link dev-sh>
chmod +x install.sh
./install.sh <aws signed link dev connector>

```

i.e
```
export id=containtel:a4ae70c4-b7ff-40c8-83c1-1e10ee166241
wget -O install.sh  "https://arduino-tmp.s3.amazonaws.com/arduino-connector.sh?AWSAccessKeyId=ASIAJJFZDTIGHJCWMGQA&Expires=1529771794&x-amz-security-token=FQoDYXdzEBoaDD8duZwY18MeYFd3CyLPAjxH7ijRrTBwduS9r8Dqm06%2BT%2B6p57cOU4I1Bn3d09lMVjPi4dhNQboAxLnYSI%2BNqxUo%2BbgNDxRbIVxzgvGWQHw7Seepjniy%2FvCKpR7DuxyNe%2B5DxA15O1fGZDQkqadxlky5jkXk1Vn9TBtGa4NCRMgIoatRBtkHI7XKpouWNYhh2jYo7ezeDRQO3m1WR7WieqVlh%2BdscL0NevGGMOh3MYf5Wsm069GuA31FmTslp3SaChf7Mq7uOI5X9XIu%2B9kcWnxXoo7dMCk5Ixq5WLkB%2BUlTt6iL4bxK7FKdlT%2FUsf5DSfBcCGwcyI2nBuFB6yjPeS5AAm0ZUU6DaEd9KUc8Fxq9M1tEQ3DnjGnKZcbaOU%2FGWw7bnOPhLcl6eiNIOtZxsvZ4MCTY3YUnO4rna4fVNScjIqMwNdb8psFarGH1Gn0e4DRNt22LFshjGZdNi01RKI%2BFqtkF&Signature=jI00Smxp33Y72ijdRJsXMIYx9h0%3D"
chmod +x install.sh
./install.sh "https://arduino-tmp.s3.amazonaws.com/arduino-connector?AWSAccessKeyId=ASIAJJFZDTIGHJCWMGQA&Expires=1529771799&x-amz-security-token=FQoDYXdzEBoaDD8duZwY18MeYFd3CyLPAjxH7ijRrTBwduS9r8Dqm06%2BT%2B6p57cOU4I1Bn3d09lMVjPi4dhNQboAxLnYSI%2BNqxUo%2BbgNDxRbIVxzgvGWQHw7Seepjniy%2FvCKpR7DuxyNe%2B5DxA15O1fGZDQkqadxlky5jkXk1Vn9TBtGa4NCRMgIoatRBtkHI7XKpouWNYhh2jYo7ezeDRQO3m1WR7WieqVlh%2BdscL0NevGGMOh3MYf5Wsm069GuA31FmTslp3SaChf7Mq7uOI5X9XIu%2B9kcWnxXoo7dMCk5Ixq5WLkB%2BUlTt6iL4bxK7FKdlT%2FUsf5DSfBcCGwcyI2nBuFB6yjPeS5AAm0ZUU6DaEd9KUc8Fxq9M1tEQ3DnjGnKZcbaOU%2FGWw7bnOPhLcl6eiNIOtZxsvZ4MCTY3YUnO4rna4fVNScjIqMwNdb8psFarGH1Gn0e4DRNt22LFshjGZdNi01RKI%2BFqtkF&Signature=BTsZzRhHnf%2Fl%2BWsXfJ9MB1ir318%3D"

```

## run integration tests with vagrant
please note that:
* the thing `devops-test:c4d6adc7-a2ca-43ec-9ea6-20568bf407fc`
* the iot IAM policy `DevicePolicy`
* the arduino user `devops-test`
* the s3 bucket `arduino-tmp`
* the test sketch `sketch_devops_integ_test`
* the private image `private_image`
are resources that must be manually created in the Arduino Cloud environment, in order to replicate the testing, you will need to create those resources on your environment and edit the test setup/teardown scripts:
* `upload_dev_artifacts_on_s3.sh`
* `create_iot_device.sh`
* `teardown_dev_artifacts.sh`
* `teardown_iot_device.sh`

In order to launch the integration test in a CI fashion do the following:
1. install vagrant from upstream link https://www.vagrantup.com/downloads.html
2. export the arduino user credentials

```
export CONNECTOR_USER=aaaaaaaa
export CONNECTOR_PASS="bbbbbb"
export CONNECTOR_PRIV_USER="cccccc"
export CONNECTOR_PRIV_PASS="ddddd"
export CONNECTOR_PRIV_IMAGE="<priv-registry-url>/<image>"
```

3. launch `make test`
4. profit

the `test` recipe:
1. spins up a ubuntu machine
2. installing your local s3 artifact after uploading it to s3 (to emulate the user install)
3. creates certs and keys on aws iot in order to talk with the connector instance in the vagrant vm
4. launch gotests (that basically do mqtt command -> vagrant ssh to check the result in the vm)
5. teardowns the aws iot things and perform all generated code and vm cleaning up
this recipe has the purpose to be used in a CI/CD context

The `test` recipe is split in 3 parts (`setup-test integ-test teardown-test`) that can be used separately to do TDD in this way:
1. launch `make setup-test`
2. write test and code
3. export the arduino user credentials

```
export CONNECTOR_USER=aaaaaaaa
export CONNECTOR_PASS="bbbbbb"
export CONNECTOR_PRIV_USER="cccccc"
export CONNECTOR_PRIV_PASS="ddddd"
export CONNECTOR_PRIV_IMAGE="<priv-registry-url>/<image>"
```

4. launch `make integ-test` all the times you need
5. launch `make teardown-test` when finished
