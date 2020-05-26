

<div align=left><a href="https://github.com/pion/ion/wiki">
    <img src="docs/imgs/ion.png" width = 15% align = "left">
</a>



#### *ION is a distributed real-time communication system, the goal is to chat anydevice, anytime, anywhere!*

![MIT](https://img.shields.io/badge/License-MIT-yellow.svg)[![Build Status](https://travis-ci.com/pion/ion.svg?branch=master)](https://travis-ci.com/pion/ion)[![Go Report Card](https://goreportcard.com/badge/github.com/pion/ion)![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/pion/ion)![GitHub tag (latest SemVer pre-release)](https://img.shields.io/github/v/tag/pion/ion?include_prereleases)](https://goreportcard.com/report/github.com/pion/ion)![Docker Pulls](https://img.shields.io/docker/pulls/pionwebrtc/ion-biz?style=plastic)[![Financial Contributors on Open Collective](https://opencollective.com/pion-ion/all/badge.svg?label=financial+contributors)](https://opencollective.com/pion-ion) ![GitHub contributors](https://img.shields.io/github/contributors-anon/pion/ion)![Twitter Follow](https://img.shields.io/twitter/follow/_PION?style=social)[![slack](https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=brightgreen)](https://pion.ly/slack)

## Architecture

<img src="https://github.com/pion/ion/raw/master/docs/imgs/arch.png" width = 100%>

## Modules


| **Name**                                                     | **Information**                                              |
| ------------------------------------------------------------ | ------------------------------------------------------------ |
| <a href="https://github.com/pion/ion"><img src="docs/imgs/go.png" height = 12% width = 10%> </a>[**ION-BIZ**](https://github.com/pion/ion)                   | *Business signal server*                                  |
| <a href="https://github.com/pion/ion"><img src="docs/imgs/go.png" height = 12% width = 10%> </a>[**ION-ISLB**](https://github.com/pion/ion)                  | *Intelligent-Server-Load-Balancing server*                 |
| <a href="https://github.com/pion/ion"> <img src="docs/imgs/go.png" height = 12% width = 10%> </a>[**ION-SFU**](https://github.com/pion/ion)                   | *Selective-Forwarding-Unit server*                         |
| <a href="https://github.com/pion/ion-sdk-js"> <img src="docs/imgs/ts.png" height = 12% width = 10%> </a> [**ION-SDK-JS**](https://github.com/pion/ion-sdk-js)         | *Ion js sdk written by typescript*                         |
| <a href="https://github.com/pion/ion-sdk-flutter"> <img src="docs/imgs/flutter.png" height = 12% width = 10%> </a>  [**ION-SDK-FLUTTER**](https://github.com/pion/ion-sdk-flutter) | *Ion flutter sdk powered by [flutter-webrtc](https://github.com/cloudwebrtc/flutter-webrtc)* |
| <a href="https://github.com/pion/ion-app-web"> <img src="docs/imgs/chrome.png" height = 12% width = 10%> </a> [**ION-APP-WEB**](https://github.com/pion/ion-app-web)       | *Ion web app*                                              |
| <a href="https://github.com/pion/ion-app-flutter"> <img src="docs/imgs/flutter.png" height = 12% width = 10%> </a> [**ION-APP-FLUTTER**](https://github.com/pion/ion-app-flutter) | *Ion flutter app*                                          |

## Documentation

### Deps

This project uses docker

https://docs.docker.com/get-docker/

### Setup
```
docker network create ionnet
```

### Run
```
docker-compose -f docker-compose.stable.yml up
```

For dev and more options see the wiki

* [Development](https://github.com/pion/ion/wiki/DockerDev)


## Roadmap

[Projects](https://github.com/pion/ion/projects/1)

## Maintainers

<a href="https://github.com/adwpc"><img width="60" height="60" src="https://github.com/adwpc.png?size=500"/></a><a href="https://github.com/cloudwebrtc"><img width="60" height="60" src="https://github.com/cloudwebrtc.png?size=500"/></a><a href="https://github.com/kangshaojun"><img width="60" height="60" src="https://github.com/kangshaojun.png?size=500"/></a><a href="https://github.com/tarrencev"><img width="60" height="60" src="https://github.com/tarrencev.png?size=500"/></a><a href="https://github.com/jbrady42"><img width="60" height="60" src="https://github.com/jbrady42.png?size=500"/></a>

## Contributors

<a href="https://github.com/pion/ion/graphs/contributors"><img src="https://opencollective.com/pion-ion/contributors.svg?width=890&button=false" /></a>

*Original Author: [adwpc](https://github.com/adwpc) [cloudwebrtc](https://github.com/cloudwebrtc)*

*Community Hero: [Sean-Der](https://github.com/Sean-Der)*



