# ION

ION is a distributed RTC system written by pure go and flutter

[![Financial Contributors on Open Collective](https://opencollective.com/pion-ion/all/badge.svg?label=financial+contributors)](https://opencollective.com/pion-ion) [![Build Status](https://travis-ci.com/pion/ion.svg?branch=master)](https://travis-ci.com/pion/ion)
![MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
[![slack](https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=brightgreen)](https://pion.ly/slack)
[![Go Report Card](https://goreportcard.com/badge/github.com/pion/ion)](https://goreportcard.com/report/github.com/pion/ion)

## Architecture

![ion](docs/imgs/cloud.png)

## Features

- [x] Distributed Node

  - [x] Standalone BIZ/ISLB and SFU node
  - [x] Message Queue by NATS
  - [x] SFU by Pure GO
  - [x] MCU (WIP)
  - [x] SFU<-->SFU relay (WIP)
  - [x] High Performance (WIP)

  - [x] Media Streaming
    - [x] WebRTC stack
    - [x] SIP stack (WIP)
    - [x] RTP/RTP over KCP
    - [x] JitterBuffer
      - [x] Nack
      - [x] PLI
      - [x] Lite-REMB
      - [x] Transport-CC(WIP)
      - [x] Anti-Loss-Package 30%+

- [x] SDK
  - [x] Flutter SDK
  - [x] JS SDK
- [x] Demo

## Contributing

- [adwpc](https://github.com/adwpc) - _Original Author - ion server_
- [cloudwebrtc](https://github.com/cloudwebrtc) - _Original Author - ion server and client sdk_
- [kangshaojun](https://github.com/kangshaojun) - _Contributor UI - flutter and react.js_

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
#non-docker
./scripts/installDeps.sh

#docker
Building is not required, pre-made images are hosted
```

### 3. run

```
#non-docker
./scripts/allRestart.sh

#docker
docker-compose up
```

### 4. let's chat

Open this url with chrome

```
https://yourip:8080
```
