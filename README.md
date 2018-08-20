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

#### Get data for a single package

```
{"package": "firmware-linux"}
--> $aws/things/{{id}}/apt/get/post

INFO: {"packages":[
        {"Name":"firmware-linux","Status":"not-installed","Architecture":"","Version":""}
    ]}
<-- $aws/things/{{id}}/apt/get
```

The response is an array that may have 1 element if the package is found or 0 if no packages are found.

#### Search for installed/installable/upgradable packages

```
{"search": "linux"}
--> $aws/things/{{id}}/apt/list/post

INFO: {"packages":[
        {"Name":"binutils-x86-64-linux-gnu","Status":"installed","Architecture":"amd64","Version":"2.29.1-4ubuntu1"},
        {"Name":"firmware-linux","Status":"not-installed","Architecture":"","Version":""},
        ...
    ],"page":0,"pages":6,"total_items":182}
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

#### Heartbeat

The connector will send keep-alive messages on the following queue every 15 seconds

```
INFO: 162653.88
<-- $aws/things/{{id}}/heartbeat
```

The number is the uptime in seconds

### Containers Management

#### Containers ps

implements ```docker ps -a``` and gives back the docker api response transparently

```
{}
--> $aws/things/{{id}}/containers/ps/post

INFO: [
  {
    "Command": "docker-entrypoint.sh redis-server",
    "Created": 1527232692,
    "HostConfig": {
      "NetworkMode": "default"
    },
    "Id": "019e3a2f50d24c81a67847f92e23e79f0c0056a210fa4b0d1bf964f9db71680f",
    "Image": "docker.io/library/redis",
    "ImageID": "sha256:bfcb1f6df2db8a62694aaa732a3133799db59c6fec58bfeda84e34299e7270a8",
    "Labels": {},
    "Mounts": [
      {
        "Destination": "/data",
        "Driver": "local",
        "Mode": "",
        "Name": "6cb1395830bd65cfac62dd55d4ed19499911191a92a759b2410250608f5df6f0",
        "Propagation": "",
        "RW": true,
        "Source": "",
        "Type": "volume"
      }
    ],
    "Names": [
      "/fabrizio-redis"
    ],
    "NetworkSettings": {
      "Networks": {
        "bridge": {
          "Aliases": null,
          "DriverOpts": null,
          "EndpointID": "4cc1922ef56f401668ee745d3e819cc21b804dfc119b2c4928e3819175522f66",
          "Gateway": "172.17.0.1",
          "GlobalIPv6Address": "",
          "GlobalIPv6PrefixLen": 0,
          "IPAMConfig": null,
          "IPAddress": "172.17.0.2",
          "IPPrefixLen": 16,
          "IPv6Gateway": "",
          "Links": null,
          "MacAddress": "02:42:ac:11:00:02",
          "NetworkID": "8459eb5e58305845527f1e6c737c059f6b409ef581ec8f0b168c1efe67304887"
        }
      }
    },
    "Ports": [
      {
        "PrivatePort": 6379,
        "Type": "tcp"
      }
    ],
    "State": "running",
    "Status": "Up 6 hours"
  },
  ...
]
<-- $aws/things/{{id}}/containers/ps/post
```

#### Containers Images

implements ```docker images``` and gives back the docker api response transparently
 
```
{}
--> $aws/things/{{id}}/containers/images/post

INFO: [
    {
    "Containers": -1,
    "Created": 1527112010,
    "Id": "sha256:316536b3f5c4aa1102f4ca80282c4d65b6f8da3a267489887b9d837660c7b19b",
    "Labels": null,
    "ParentId": "",
    "RepoDigests": [
      "postgres@sha256:db7a4b960bfbe98ad94799f4a00a4203284c9804177ba317e1f9829fa1237632"
    ],
    "RepoTags": [
      "postgres:latest"
    ],
    "SharedSize": -1,
    "Size": 235337390,
    "VirtualSize": 235337390
  },
  ...
]
<-- $aws/things/{{id}}/containers/images/post
```


#### Containers Action

```docker run <image>``` 

```
{
  "action": "run",
  "background": true,
  "image": "redis",
  "name": "my-redis-container"
}
--> $aws/things/{{id}}/containers/action/post

INFO: [
  {
  "action": "run",
  "background": true,
  "id": "316536b3f5c4aa1102f4ca80282c4d65b6f8da3a267489887b9d837660c7b19b",
  "image": "redis",
  "name": "my-redis-container"
 }
]
<-- $aws/things/{{id}}/containers/action/post
```


```docker start <container-id>``` 

```
{
  "action": "start",
  "background": true,
  "id": "316536b3f5c4aa1102f4ca80282c4d65b6f8da3a267489887b9d837660c7b19b",
}
--> $aws/things/{{id}}/containers/action/post

INFO: [
  {
  "action": "start",
  "background": true,
  "id": "316536b3f5c4aa1102f4ca80282c4d65b6f8da3a267489887b9d837660c7b19b",
  "image": "redis",
  "name": "my-redis-container"
 }
]
<-- $aws/things/{{id}}/containers/action/post
```

```docker stop <container-id>``` 

```
{
  "action": "stop",
  "id": "316536b3f5c4aa1102f4ca80282c4d65b6f8da3a267489887b9d837660c7b19b",
}
--> $aws/things/{{id}}/containers/action/post

INFO: [
  {
  "action": "stop",
  "background": true,
  "id": "316536b3f5c4aa1102f4ca80282c4d65b6f8da3a267489887b9d837660c7b19b",
  "image": "redis",
  "name": "my-redis-container"
 }
]
<-- $aws/things/{{id}}/containers/action/post
```

```docker remove <container-id>``` and ```docker image prune -a```
```
{
  "action": "remove",
  "id": "316536b3f5c4aa1102f4ca80282c4d65b6f8da3a267489887b9d837660c7b19b",
}
--> $aws/things/{{id}}/containers/action/post

INFO: [
  {
  "action": "remove",
  "background": true,
  "id": "316536b3f5c4aa1102f4ca80282c4d65b6f8da3a267489887b9d837660c7b19b",
  "image": "redis",
  "name": "my-redis-container"
 }
]
<-- $aws/things/{{id}}/containers/action/post
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

## Generate temporary installer script
```
aws-google-auth -p arduino       
go build -ldflags "-X main.version=2.0.21" github.com/arduino/arduino-connector
aws --profile arduino s3 cp arduino-connector-dev.sh s3://arduino-tmp/arduino-connector.sh
aws s3 presign --profile arduino s3://arduino-tmp/arduino-connector.sh --expires-in $(expr 3600 \* 72)
#use this link i the wget of the getting started script
aws --profile arduino s3 cp arduino-connector s3://arduino-tmp/
aws s3 presign --profile arduino s3://arduino-tmp/arduino-connector  --expires-in $(expr 3600 \* 72)
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

## run tests with vagrant
1. install vagrant from upstream link https://www.vagrantup.com/downloads.html
2. `cd test`
3. `vagrant up`
4. `vagrant provision`
5. profit
