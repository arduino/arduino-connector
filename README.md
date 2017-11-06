# Arduino Connector

The Arduino Connector allows your device to connect to the Arduino Cloud, and push and receive message through the [MQTT protocol](http://mqtt.org/). You can see and control all your cloud-enabled devices via a web app called [My Devices](https://create.arduino.cc/devices).

## Install

Follow the Getting Started guides to install the connector and allow your devices to communincate with the cloud via Arduino Create. You can install the connector onto a [Up2 board](https://create.arduino.cc/getting-started/up2) or a generic [Intel-based platform running Linux](https://create.arduino.cc/getting-started/intel-platforms).

### How does it work?

The Arduino Connector gets installed on a device and does the following things:

- Connects to MQTT using the certificate and key generated during installation
- Starts and Stops sketches according to the received commands from MQTT
- Collects the output of the sketches in order to send them on MQTT

### Install

The Arduino Connector is tied to a specific device registered within the Arduino Cloud. The [getting started guide](https://create.arduino.cc/getting-started) does everything for you.

Make sure you have an Arduino Account and you are able to log at: https://auth.arduino.cc/login

Please write us at auth@arduino.cc if you encounter any issue loggin in and you need support.