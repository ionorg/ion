<h1 align="center">
  <img src="docs/imgs/ion.jpg" alt="Ion" height="250px">
  <br>
  Ion
  <br>
</h1>
<h4 align="center">A distributed RTC platform written in pure Go</h4>
<p align="center">
  <a href="https://opencollective.com/pion-ion"><img src="https://opencollective.com/pion-ion/all/badge.svg?label=financial+contributors" alt="Ion Open Collective"></a>
  <a href="https://pion.ly/slack"><img src="https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=brightgreen" alt="Slack Widget"></a>
  <a href="https://travis-ci.org/pion/webrtc"><img src="https://travis-ci.org/pion/webrtc.svg?branch=master" alt="Build Status"></a>
  <a href="https://goreportcard.com/badge/github.com/pion/ion"><img src="https://goreportcard.com/badge/github.com/pion/ion" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
</p>
<br>

## Wiki

https://github.com/pion/ion/wiki

## Architecture

![arch](https://github.com/pion/ion/raw/master/docs/imgs/arch.png)

## SDKs

[ion-sdk-js](https://github.com/pion/ion-sdk-js) contains a frontend typescript sdk.

[ion-sdk-flutter](https://github.com/pion/ion-sdk-flutter) contains a frontend flutter sdk.

[ion.py](https://github.com/pion/ion.py) contains a service discovery library for creating python services.

## Applications

[ion-app-web](https://github.com/pion/ion-app-web) contains a frontend web application written in javascript.

[ion-app-flutter](https://github.com/pion/ion-app-flutter) contains a frontend web/iOS/Android application written in flutter.

## Usage

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

#### 5. Frontend

Head over to [Ion Web App](https://github.com/pion/ion-app-web) to bring up the front end.

The web app repo also contains examples of exposing the ion biz websocket via reverse proxy with automatic SSL.

## Roadmap

[Projects](https://github.com/pion/ion/projects/1)


## Contributing
We welcome contributions to Ion!

- [adwpc](https://github.com/adwpc) - _Original Author - ion server_
- [cloudwebrtc](https://github.com/cloudwebrtc) - _Original Author - ion server and client sdk_
- [kangshaojun](https://github.com/kangshaojun) - _Contributor UI - flutter and react.js_
- [Sean-Der](https://github.com/Sean-Der) - _ion server and docker file_
- [sashaaro](https://github.com/sashaaro) - _docker file_
- [tarrencev](https://github.com/tarrencev) - _audio video process_
- [jbrady42](https://github.com/jbrady42) - _load testing, performance improvements, bug fixes_
