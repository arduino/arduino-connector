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

### API

To control the arduino-connector you must have:

- the ID of the device in which the arduino-connector has been installed (eg `username:0002251d-4e19-4cc8-a4a9-1de215bfb502`)
- a working mqtt connection

Send messages to the topic ending with /post, Receive the answer from the topic ending with /. Errors are sent to the same endpoint.

You can distinguish between errors and non-errors because of the INFO: or ERROR: prefix of the message

### Status

Retrieve the status of the connector
```
{}
--> $aws/things/{{id}}/status/post

INFO: {
    "sketches": {
        "4c1f3a9d-ed78-4ae4-94c8-bcfa2e94c692": {
            "name":"sketch_oct31a",
            "id":"4c1f3a9d-ed78-4ae4-94c8-bcfa2e94c692",
            "pid":31343,
            "status":"RUNNING",
            "endpoints":null
        }
    }
}
<-- $aws/things/{{id}}/status
```

### Upload a sketch on the connector

```
{
  "token": "toUZDUNTcooVlyqAUwooBGAEtgr8iPzp017RhcST8gM.bDBgrxVzKKySBX-kBPMRqFRqlP3j_cwlgt9qPh_Ct2Y",
  "url": "https://api-builder.arduino.cc/builder/v1/compile/sketch_oct31a.bin",
  "name": "sketch_oct31a",
  "id": "4c1f3a9d-ed78-4ae4-94c8-bcfa2e94c692"
}
--> $aws/things/{{id}}/upload/post

INFO: Sketch started with PID 570
<-- $aws/things/{{id}}/upload
```

### Update the arduino-connector (doesn't return anything)

```
{
  "url": "https://downloads.arduino.cc/tools/arduino-connector",
}
--> $aws/things/{{id}}/update/post

<-- $aws/things/{{id}}/sketch
```

### Update the arduino-connector

```
{
  "url": "https://downloads.arduino.cc/tools/arduino-connector",
}
--> $aws/things/{{id}}/update/post

<-- $aws/things/{{id}}/sketch
```

### Retrieve the stats of the machine (memory, disk, networks)

```
{}
--> $aws/things/{{id}}/stats/post

INFO: {
   "memory":{
      "FreeMem":1317964,
      "TotalMem":15859984,
      "AvailableMem":8184204,
      "Buffers":757412,
      "Cached":6569888,
      "FreeSwapMem":0,
      "TotalSwapMem":0
   },
   "disk":[
        {
            "Device":"sysfs",
            "Type":"sysfs",
            "MountPoint":"/sys",
            "FreeSpace":0,
            "AvailableSpace":0,
            "DiskSize":0
        },
    ],
   "network":{
      "Devices":[
         {
            "AccessPoints":[
               {
                  "Flags":1,
                  "Frequency":2437,
                  "HWAddress":"58:6D:8F:8F:FD:F3",
                  "MaxBitrate":54000,
                  "Mode":"Nm80211ModeInfra",
                  "RSNFlags":392,
                  "SSID":"ssid-2g",
                  "Strength":80,
                  "WPAFlags":0
               }
            ],
            "AvailableConnections":[
               {
                  "802-11-wireless":{
                     "mac-address":"QOIwy+Ef",
                     "mac-address-blacklist":[],
                     "mode":"infrastructure",
                     "security":"802-11-wireless-security",
                     "seen-bssids":[
                        "58:6D:8F:8F:FD:F3"
                     ],
                     "ssid":"QkNNSWxhYnMtMmc="
                  },
                  "802-11-wireless-security":{
                     "auth-alg":"open",
                     "group":[],
                     "key-mgmt":"wpa-psk",
                     "pairwise":[],
                     "proto":[]
                  },
                  "connection":{
                     "id":"ssid-2g",
                     "permissions":[],
                     "secondaries":[],
                     "timestamp":1513953989,
                     "type":"802-11-wireless",
                     "uuid":"b5dd1024-db02-4e0f-ad3b-c41c375f750a"
                  },
                  "ipv4":{
                     "address-data":[],
                     "addresses":[],
                     "dns":[],
                     "dns-search":[],
                     "method":"auto",
                     "route-data":[],
                     "routes":[]
                  },
                  "ipv6":{
                     "address-data":[],
                     "addresses":[],
                     "dns":[],
                     "dns-search":[],
                     "method":"auto",
                     "route-data":[],
                     "routes":[]
                  }
               }
            ],
            "DeviceType":"NmDeviceTypeWifi",
            "IP4Config":{
               "Addresses":[
                  {
                     "Address":"10.130.22.132",
                     "Prefix":24,
                     "Gateway":"10.130.22.1"
                  }
               ],
               "Domains":[],
               "Nameservers":[
                  "10.130.22.1"
               ],
               "Routes":[]
            },
            "Interface":"wlp4s0",
            "State":"NmDeviceStateActivated"
         }
      ],
      "Status":"NmStateConnectedGlobal"
   }
}
<-- $aws/things/{{id}}/stats
```

### Configure the wifi (doesn't return anything)

```
{
  "ssid": "ssid-2g",
  "password": "passwordssid"
}
--> $aws/things/{{id}}/stats/post

<-- $aws/things/{{id}}/stats
```

### Package Management

#### Retrieve a list of the upgradable packages

```
{}
--> $aws/things/{{id}}/apt/list/post

INFO: {"packages":[
        {"Name":"firefox","Status":"upgradable","Architecture":"amd64","Version":"57.0.3+build1-0ubuntu0.17.10.1"},
        {"Name":"firefox-locale-en","Status":"upgradable","Architecture":"amd64","Version":"57.0.3+build1-0ubuntu0.17.10.1"}
    ],
    "page":0,"pages":1}
<-- $aws/things/{{id}}/apt/list
```

#### Search for installed/installable/upgradable packages

```
{"search": "linux"}
--> $aws/things/{{id}}/apt/list/post

INFO: {"packages":[
        {"Name":"binutils-x86-64-linux-gnu","Status":"installed","Architecture":"amd64","Version":"2.29.1-4ubuntu1"},
        {"Name":"firmware-linux","Status":"not-installed","Architecture":"","Version":""},
        ...
    ],"page":0,"pages":6}
<-- $aws/things/{{id}}/apt/list
```

Navigate pages

```
{"search": "linux", "page": 2}
--> $aws/things/{{id}}/apt/list/post

INFO: {"packages":[
        {"Name":"linux-image-4.10.0-30-generic","Status":"config-files","Architecture":"amd64","Version":"4.10.0-30.34"},
        {"Name":"linux-image-4.13.0-21-generic","Status":"installed","Architecture":"amd64","Version":"4.13.0-21.24"},
        ...
    ],"page":2,"pages":6}
<-- $aws/things/{{id}}/apt/list
```

#### Update the list of available packages
```
{}
--> $aws/things/{{id}}/apt/update/post

INFO: {
    "output" : "apt command output..."
}
<-- $aws/things/{{id}}/apt/update/post
```

#### Install a set of packages

```
{"packages" : { "package-a", "package-b", .... }}
--> $aws/things/{{id}}/apt/install/post

INFO: {
    "output" : "apt command output..."
}
<-- $aws/things/{{id}}/apt/install/post
```

#### Upgrade a set of packages

```
{"packages" : { "package-a", "package-b", .... }}
--> $aws/things/{{id}}/apt/upgrade/post

INFO: {
    "output" : "apt command output..."
}
<-- $aws/things/{{id}}/apt/upgrade/post
```

#### Upgrade all packages

```
{"packages" : { }}
--> $aws/things/{{id}}/apt/upgrade/post

INFO: {
    "output" : "apt command output..."
}
<-- $aws/things/{{id}}/apt/upgrade/post
```

#### Uninstall a set of packages

```
{"packages" : { "package-a", "package-b", .... }}
--> $aws/things/{{id}}/apt/remove/post

INFO: {
    "output" : "apt command output..."
}
<-- $aws/things/{{id}}/apt/remove/post
```

### Repositories management

The following API handles repositories, each repository is
repesented by the following JSON structure:

```
{
    "enabled":      true/false,
	"sourceRepo":   true/false,
	"options":      "...",
	"uri":          "...",
	"distribution": "...",
	"components":   "...",
	"comment":      "...",
}
```

for clarity, in the following descriptions, we will refer to the above structure with the `REPOSITORYxx` shortcut.

#### List repositories

```
{}
--> $aws/things/{{id}}/apt/repos/list/post

INFO: {
    REPOSITORY1,
    REPOSITORY2,
    ....
}
<-- $aws/things/{{id}}/apt/repos/list/post
```

#### Add repository

```
{ "repository" : REPOSITORY1 }
--> $aws/things/{{id}}/apt/repos/add/post

INFO: OK
<-- $aws/things/{{id}}/apt/repos/add/post
```

#### Remove repository

```
{ "repository" : REPOSITORY1 }
--> $aws/things/{{id}}/apt/repos/remove/post

INFO: OK
<-- $aws/things/{{id}}/apt/repos/remove/post
```

#### Edit repository

The repository in `old_repository` is replaced with `new_repository`

```
{
    "old_repository": REPOSITORY1,
    "new_repository": REPOSITORY2,
}
--> $aws/things/{{id}}/apt/repos/edit/post

INFO: OK
<-- $aws/things/{{id}}/apt/repos/edit/post
```

## Compile
```
go get github.com/arduino/arduino-connector
go build -ldflags "-X main.version=$VERSION" github.com/arduino/arduino-connector
```

## Autoupdate
```
go get github.com/sanbornm/go-selfupdate
./bin/go-selfupdate arduino-connector $VERSION
# scp -r public/* user@server:/var/www/files/arduino-connector
```

