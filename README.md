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
- [Sean-Der](https://github.com/Sean-Der) - _ion server and docker_

## Roadmap

[Projects](https://github.com/pion/ion/projects/1)
Welcome contributing to ion!

## Stars

[![Stargazers over time](https://starchart.cc/pion/ion.svg)](https://starchart.cc/pion/ion)

# Screenshots

## iOS/Android

<img width="180" height="370" src="screenshots/flutter/flutter-01.jpg"/> <img width="180" height="370" src="screenshots/flutter/flutter-02.jpg"/> <img width="180" height="370" src="screenshots/flutter/flutter-03.jpg"/>

## PC/HTML5

<img width="360" height="265" src="screenshots/web/ion-01.jpg"/> <img width="360" height="265" src="screenshots/web/ion-02.jpg"/>
<img width="360" height="265" src="screenshots/web/ion-04.jpg"/> <img width="360" height="265" src="screenshots/web/ion-05.jpg"/>

## How to use

### Local Deployment
#### 1. clone
```
git clone https://github.com/pion/ion
```

#### 2. run
Firstly pull images. Skip this command if you want build images locally
```
docker-compose pull
```

```
docker-compose up
```

#### 3. chat
Open this url with chrome

```
http://localhost:8080
```

### Online Deployment

#### 1. clone

```
git clone https://github.com/pion/ion
```

#### 2. set env

```
export WWW_URL=yourdomain
export ADMIN_EMAIL=yourname@yourdomain
```

#### 3. run

```
docker-compose up
```

#### 3. chat

Open this url with chrome

```
https://yourdomain:8080
```

### Docker Tips

The provided docker-compose works for deploying to open usage, and can also be used for local development. It also supports auto-generate of certificates via LetsEncrypt.

It accepts the following enviroment variables.

* `WWW_URL` -- Public URL if auto-generating certificates
* `ADMIN_EMAIL`  -- Email if auto-generating certificates

To run on `conference.pion.ly` you would run `WWW_URL=conference.pion.ly ADMIN_EMAIL=admin@pion.ly docker-compose up`

If `WWW_URL` is set you will access via `https://yourip:8080` OR `http://yourip:8080` if not running with TLS enabled.