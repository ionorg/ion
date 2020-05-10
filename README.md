# ION

ION is a distributed RTC system written by pure go and flutter

[![Financial Contributors on Open Collective](https://opencollective.com/pion-ion/all/badge.svg?label=financial+contributors)](https://opencollective.com/pion-ion) [![Build Status](https://travis-ci.com/pion/ion.svg?branch=master)](https://travis-ci.com/pion/ion)
![MIT](https://img.shields.io/badge/License-MIT-yellow.svg)
[![slack](https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=brightgreen)](https://pion.ly/slack)
[![Go Report Card](https://goreportcard.com/badge/github.com/pion/ion)](https://goreportcard.com/report/github.com/pion/ion)

### Notice: Please use v0.3.0, master is not stable now


## Wiki

<img src="docs/imgs/ion.jpg" width = "10%" />https://github.com/pion/ion/wiki

## Architecture

![arch](https://github.com/pion/ion/raw/master/docs/imgs/arch.png)

## Contributor

- [adwpc](https://github.com/adwpc) - _Original Author - ion server_
- [cloudwebrtc](https://github.com/cloudwebrtc) - _Original Author - ion server and client sdk_
- [kangshaojun](https://github.com/kangshaojun) - _Contributor UI - flutter and react.js_
- [Sean-Der](https://github.com/Sean-Der) - _ion server and docker file_
- [sashaaro](https://github.com/sashaaro) - _docker file_
- [tarrencev](https://github.com/tarrencev) - _audio video process_

## SDK

[ion-sdk-js](https://github.com/pion/ion-sdk-js)

[ion-sdk-flutter](https://github.com/pion/ion-sdk-flutter)

## APP

[ion-app-web](https://github.com/pion/ion-app-web)

[ion-app-flutter](https://github.com/pion/ion-app-flutter)

# Screenshots

## iOS/Android

<img width="180" height="370" src="screenshots/flutter/flutter-01.jpg"/> <img width="180" height="370" src="screenshots/flutter/flutter-02.jpg"/> <img width="180" height="370" src="screenshots/flutter/flutter-03.jpg"/>

## PC/HTML5

<img width="360" height="265" src="screenshots/web/ion-01.jpg"/> <img width="360" height="265" src="screenshots/web/ion-02.jpg"/>
<img width="360" height="265" src="screenshots/web/ion-04.jpg"/> <img width="360" height="265" src="screenshots/web/ion-05.jpg"/>

## How to use

Docker commands require the ionnet docker network

First run:
```
docker network create ionnet
```

### Deployment

#### 1. Clone
```
git clone https://github.com/pion/ion
```

#### 2. Setup
Firstly pull images. Skip this command if you want build images locally
```
docker-compose pull
```

#### 3. Run
```
docker-compose up
```

#### 4. Expose Ports
(Skip if only exposing locally)

Ensure the following ports are exposed or forwarded.

```
5000-5200/udp
```

#### 5. Chat

Head over to [Ion Web App](https://github.com/pion/ion-app-web) to bring up the front end.

The web app repo also contains examples of exposing the ion biz websocket via reverse proxy with automatic SSL.

## Roadmap

[Projects](https://github.com/pion/ion/projects/1)
Welcome contributing to ion!
