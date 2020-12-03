
<div align=left><a href="https://github.com/pion/ion/wiki">
    <img src="https://github.com/pion/ion/raw/master/docs/imgs/ion.png" width = 15% align = "left">
</a>

#### *ION is a distributed real-time communication system, the goal is to chat anydevice, anytime, anywhere!*

![MIT](https://img.shields.io/badge/License-MIT-yellow.svg)[![Go Report Card](https://goreportcard.com/badge/github.com/pion/ion)![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/pion/ion)![GitHub tag (latest SemVer pre-release)](https://img.shields.io/github/v/tag/pion/ion?include_prereleases)](https://goreportcard.com/report/github.com/pion/ion)![Docker Pulls](https://img.shields.io/docker/pulls/pionwebrtc/ion-biz?style=plastic)[![Financial Contributors on Open Collective](https://opencollective.com/pion-ion/all/badge.svg?label=financial+contributors)](https://opencollective.com/pion-ion) ![GitHub contributors](https://img.shields.io/github/contributors-anon/pion/ion)![Twitter Follow](https://img.shields.io/twitter/follow/_PION?style=social)[![slack](https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=brightgreen)](https://pion.ly/slack)

<br />

## Distributed Real-time Communication System

+ **ION** is a secure, self-hosted [WebRTC](https://webrtc.org/) video conferencing [SFU (what is an SFU?)](https://testrtc.com/different-multiparty-video-conferencing/), that you can host today in the cloud or on-premise.
+ This **ION** repository contains the backend cluster services, so you also need to deploy the [web app](https://github.com/pion/ion-app-web) or install a [flutter client for web, desktop or mobile](https://github.com/pion/ion-app-flutter)
+ **ION**'s mission is to deliver world-class tools for creating communication systems, and many people build their projects on top of it:
  + [**ION** Cluster](https://github.com/pion/ion) (This project!) is composed of two services [`biz` + `ISLB` (see glossary)](docs/glossary.md) and uses `NATS`, `etcd` and `redis` as databases to administer room membership, manage text chat, verify JWT authentication and assign clients to the proper SFU in a multi-datacenter architecture. **ION** Cluster also builds its own version of `ion-sfu` binary, which is lightly adapted to use `NatsRPC` for signaling (which is how `biz` and `ISLB` trade messages internally).
  + [`ion-sfu` (external)](https://github.com/pion/ion-sfu), which handles WebRTC streams, can be used as a standalone SFU for designing custom chat experiences or implementing your own scaling architecture. `ion-sfu` is equally capable of forwarding Video, Audio and DataChannel tracks, and can handle arbitrary non-media data transport.
  + [`ion-avp` Audio/Video Processing](https://github.com/pion/ion-avp) (WIP) is a sidecar utility for running realtime AV processing jobs, including `write-to-disk`, `ffmpeg` and `openCV`
  + *`ion-live` LIVE node (planned)* - A feed streaming gateway for supporting publishing to and from SIP/RTMP/HLS/RTSP endpoints
+ All built with [pion](https://pion.ly) and [golang](https://golang.org/), **ION** is [fast and efficient](docs/production/stress_test.md)
+ **ION** is [a young project](https://github.com/pion/ion/projects/2), under active development; some people run **ION** in production, but it's not for everyone (yet)
+ **ION** is a community effort and relies on volunteers like you and me!


## Roadmap

**NOV24 STATUS UPDATE**: The `ion` project has been undergoing a major restructuring for a few months! If you want to build on top of `ion` today, you should start with `ion-sfu`! You can still deploy `ion:0.4.6` as a ready-to-go conference demonstration.

![arch](https://github.com/pion/ion/raw/master/docs/imgs/ion-roadmap.png)

## ❤️Sponsor to help this awsome project go faster！🚀
(https://opencollective.com/pion-ion)

You can vote for feature if you are a sponsor.

Features: https://github.com/pion/ion/projects/2

## Quick-Start (*LOCALHOST ONLY*)

*NOTE:* Do not attempt to run this example on a VPS, it only works on `localhost`. Make sure you [read the docs](docs/production); WebRTC requires some specific network configuration for the SFU service (depending on your host), and the JavaScript `GetUserMedia()` API can only request camera access on pages with SSL (or `localhost`). If you are not running on `localhost`, you MUST configure networking for SFU and enable HTTPS for `ion-app-web`.


#### 1. Run Ion Backend Services
After cloning the folder, create a docker network (we use this so `ion-app-web` can communicate securely with the backend):
```
docker network create ionnet

docker-compose up
```

#### 3. Expose Ports

Ensure the ports `5000-5200/udp` are exposed or forwarded for the SFU service; 


#### 4. UI (optional)

Head over to [Ion Web App](https://github.com/pion/ion-app-web) to bring up the front end.

The web app repo also contains examples of exposing the ion biz websocket via reverse proxy with automatic SSL.

For dev and more options see the wiki

* [Development](https://github.com/pion/ion/tree/master/docs)



## Documentation
+ [Development](docs/dev/)
    + [Quick Start](docs/dev/quick_start.md)
    + [Docker Compose](docs/dev/docker.md)
    + [Debugging](docs/dev/debugging.md)
+ [Production](docs/production/)
    + [Docker Compose](docs/production/README.md)
    + [Kubernetes](kube/README.md)
    + [Stress Test](docs/production/stress_test.md)
+ [Server](docs/server_features.md)
+ [Clients](docs/client_features.md)
+ Ion SDKs
    + [SDK - Javascript](https://github.com/pion/ion-sdk-js)
    + [SDK - Flutter](https://github.com/pion/ion-sdk-flutter)
+ Open-Source ION Clients
    + [Ion Web App](https://github.com/pion/ion-app-web)
    + [Ion Flutter App](https://github.com/pion/ion-app-flutter)
+ Other Ion Projects
    + [Ion Load Tool](https://github.com/pion/ion-load-tool)


+ [Glossary / Definitions](docs/glossary.md)
+ [Frequently Asked Questions](docs/faq.md)

## Architecture
![arch](https://github.com/pion/ion/raw/master/docs/imgs/arch.png)

## Maintainers

<a href="https://github.com/adwpc"><img width="60" height="60" src="https://github.com/adwpc.png?size=500"/></a><a href="https://github.com/cloudwebrtc"><img width="60" height="60" src="https://github.com/cloudwebrtc.png?size=500"/></a><a href="https://github.com/kangshaojun"><img width="60" height="60" src="https://github.com/kangshaojun.png?size=500"/></a><a href="https://github.com/tarrencev"><img width="60" height="60" src="https://github.com/tarrencev.png?size=500"/></a><a href="https://github.com/jbrady42"><img width="60" height="60" src="https://github.com/jbrady42.png?size=500"/></a><a href="https://github.com/leewardbound"><img width="60" height="60" src="https://github.com/leewardbound.png?size=500"/></a><a href="https://github.com/cgojin"><img width="60" height="60" src="https://github.com/cgojin.png?size=500"/></a>

## Contributors

<a href="https://github.com/pion/ion/graphs/contributors"><img src="https://opencollective.com/pion-ion/contributors.svg?width=890&button=false" /></a>

*Original Author: [adwpc](https://github.com/adwpc) [cloudwebrtc](https://github.com/cloudwebrtc)*

*Community Hero: [Sean-Der](https://github.com/Sean-Der)*
