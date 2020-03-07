# ION

ION is a distributed RTC system written by pure go and flutter

[![Financial Contributors on Open Collective](https://opencollective.com/pion-ion/all/badge.svg?label=financial+contributors)](https://opencollective.com/pion-ion) [![Build Status](https://travis-ci.com/pion/ion.svg?branch=master)](https://travis-ci.com/pion/ion)
![MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
[![slack](https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=brightgreen)](https://pion.ly/slack)
[![Go Report Card](https://goreportcard.com/badge/github.com/pion/ion)](https://goreportcard.com/report/github.com/pion/ion)

## Architecture

![ion](docs/imgs/cloud.png)

## Features

- [x] Server
  - [x] OS
    - [x] CentOS 7
    - [x] Ubuntu 16.04.6 LTS
    - [x] macOS Mojave
  - [x] Signal
    - [x] WebSocket
  - [x] Media
    - [x] WebRTC
    - [x] RTP/RTCP
    - [x] Nack
    - [x] PLI
    - [x] Anti-Loss-Package 30%~50%
  - [x] Distributed System
    - [x] ION-ION RTP relay
    - [x] MQ support
- [x] Client
  - [x] SDK
    - [x] Flutter
    - [x] JS
  - [x] Demo

## Contributing

- [adwpc](https://github.com/adwpc) - _Original Author - ion sfu server_
- [cloudwebrtc](https://github.com/cloudwebrtc) - _Original Author - ion sfu sdk_
- [kangshaojun](https://github.com/kangshaojun) - _Contributor UI(flutter/react.js)_

## Roadmap

[Projects](https://github.com/pion/ion/projects/1)
Welcome contributing to ion!

## Project status

[![Stargazers over time](https://starchart.cc/pion/ion.svg)](https://starchart.cc/pion/ion)

# Screenshots

## iOS/Android

<img width="180" height="370" src="screenshots/flutter/flutter-01.jpg"/> <img width="180" height="370" src="screenshots/flutter/flutter-02.jpg"/> <img width="180" height="370" src="screenshots/flutter/flutter-03.jpg"/>

## PC/HTML5

<img width="360" height="265" src="screenshots/web/ion-01.jpg"/> <img width="360" height="265" src="screenshots/web/ion-02.jpg"/>
<img width="360" height="265" src="screenshots/web/ion-04.jpg"/> <img width="360" height="265" src="screenshots/web/ion-05.jpg"/>

## How to use

### 1. make key

```
./scripts/makeKey.sh
```

### 2. build

```
#docker
docker-compose build

#centos
./scripts/centos/installDeps.sh

#ubuntu
./scripts/ubuntu/installDeps.sh

#mac
./scripts/mac/installDeps.sh
```

### 3. run

```
#docker
docker-compose up

#centos
./scripts/centos/allRestart.sh

#ubuntu
./scripts/centos/allRestart.sh

#mac
./scripts/mac/allRestart.sh
```

### 4. let's chat

Open this url with chrome

```
https://yourip:8080
```
