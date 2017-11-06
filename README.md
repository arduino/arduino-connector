# Arduino Connector

The Arduino Connector allows your device to connect to the Arduino Cloud, and push and receive message through the [MQTT protocol](http://mqtt.org/). You can see and control all your cloud-enabled devices via a web app called [My Devices](https://create.arduino.cc/devices).

## Install

Follow the Getting Started guides to install the connector and allow your devices to communincate with the cloud via Arduino Create. You can install the connector onto a [Up2 board](https://create.arduino.cc/getting-started/up2) or a generic [Intel-based platform running Linux](https://create.arduino.cc/getting-started/intel-platforms).

### How does it work?

The Arduino Connector gets installed on a device and does the following things:

- Connects to MQTT using the certificate and key generated during installation
- Starts and Stops sketches according to the received commands from MQTT
- Collects the output of the sketches in order to send them on MQTT

### Install from source

The Arduino Connector is tied to a specific device registered within the Arduino Cloud. The [getting started guide](https://create.arduino.cc/getting-started) does everything for you, but if you want to do things the hard way, please read the following instructions:

1. To build the connector from source make sure you have the [go toolchain](https://golang.org/doc/install) installed and this repository cloned in `$GOPATH/src/github.com/arduino/arduino-connector.

```bash
$ go get 		# retrieves all the dependencies
$ go build 		# generates the arduino-connector executable
```

2. Make sure you have an Arduino Account and you are able to log at: https://auth.arduino.cc/login

Please write us at auth@arduino.cc if you encounter any issue loggin in and you need support.

3. The Arduino APIs use oauth2 for authentication. You don't need to know what it is, but you need to login and obtain an access token:

```bash
$ ./arduino-connector login
Insert your arduino username
awesome_user
Insert your arduino password
Access Token:  -kNkqSjymiOVtcUPUf17hroD5KK6VaCrVBd_a4ccE8o.nUD6N9E-jPm3fiTCexrnDp4n-GxfxsozidKuoQgIG9k
```

4. The Arduino APIs are REST, that means they follow a standard and are (hopefully) easy to work with. You can see the detailed documentation of the devices-api here: https://api2.arduino.cc/devices/docs

Try listing all of the devices you have registered with

```bash
$ curl -H "Authorization: Bearer -kNkqSjymiOVtcUPUf17hroD5KK6VaCrVBd_a4ccE8o.nUD6N9E-jPm3fiTCexrnDp4n-GxfxsozidKuoQgIG9k" -i https://api2.arduino.cc/devices/v1
HTTP/1.1 200 OK
Content-Type: application/vnd.arduino.device+json; type=collection
Date: Mon, 06 Nov 2017 08:50:21 GMT
Vary: Accept-Encoding
Content-Length: 3
Connection: keep-alive

[]

```

If this is the first time you are using the Arduino Connector you'll probably obtained an empty list

5. We can create a new device with a PUT request

$ curl -H "Authorization: Bearer -kNkqSjymiOVtcUPUf17hroD5KK6VaCrVBd_a4ccE8o.nUD6N9E-jPm3fiTCexrnDp4n-GxfxsozidKuoQgIG9k" -i -X PUT -d '{"name": "awesome_device"}' https://api2.arduino.cc/devices/v1
HTTP/1.1 201 Created
Content-Type: application/vnd.arduino.device+json
Date: Mon, 06 Nov 2017 08:57:04 GMT
Vary: Accept-Encoding
Content-Length: 201
Connection: keep-alive

{"href":"/devices/v1/awesome_user:7c369cb0-1105-478f-818d-24dc20eb7dfb","id":"awesome_user:7c369cb0-1105-478f-818d-24dc20eb7dfb","name":"awesome_device","user_id":"076a0d84-b9dd-442b-bb89-78fdc6d5028a"}
```

We have created a device with the name "awesome_device", and the API has assigned us an ID

6. Register the Arduino Connector with this device

```go build && ./arduino-connector -register -id awesome_user:7c369cb0-1105-478f-818d-24dc20eb7dfb -token -kNkqSjymiOVtcUPUf17hroD5KK6VaCrVBd_a4ccE8o.nUD6N9E-jPm3fiTCexrnDp4n-GxfxsozidKuoQgIG9k
Generate private key
Generate csr
Request certificate
Request mqtt url
Write conf to arduino-connector.cfg
Check successful mqtt connection
setupMQTT certificate.pem certificate.key awesome_user:7c369cb0-1105-478f-818d-24dc20eb7dfb a19g5nbe27wn47.iot.us-east-1.amazonaws.com
WARN: memstore.go:137: [store]    memorystore wiped
WARN: net.go:310: [net]      logic stopped
Setup completed
```

You should now have an arduino-connector.cfg file sitting in this folder, containing something like

``
id=awesome_user:7c369cb0-1105-478f-818d-24dc20eb7dfb
url=a19g5nbe27wn47.iot.us-east-1.amazonaws.com
http_proxy=
https_proxy=
all_proxy=
```

7. Launch the arduino-connector for real, this time

```
./arduino-connector -config ./arduino-connector.cfg                        2 ↵   docs    10:19:20
[1852] [INF] Starting nats-server version 1.0.4
[1852] [INF] Listening for client connections on 127.0.0.1:4222
[1852] [INF] Server is ready
setupMQTT certificate.pem certificate.key matteosuppo:7c369cb0-1105-478f-818d-24dc20eb7dfb a19g5nbe27wn47.iot.us-east-1.amazonaws.com
WARN: memstore.go:137: [store]    memorystore wiped
2017/11/06 10:22:30 Connected to MQTT
WARN: memstore.go:110: [store]    memorystore del: message 1 not found
WARN: memstore.go:110: [store]    memorystore del: message 3 not found
WARN: memstore.go:110: [store]    memorystore del: message 2 not found
WARN: memstore.go:110: [store]    memorystore del: message 4 not found
```
