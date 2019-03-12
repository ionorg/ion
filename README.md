# SFU

## 1. Architecture

![arch](arch.png)

## 2. Roadmap

* Media stream support
  * Peer-to-Browser (webrtc) *[WIP-adam]*
  * Peer-to-Peer (rtp)

* Singal exchange support
  * Singal protocol (room/stream/relay) *[WIP-adam]*
  * Exchange sdp/candidate/other (edge) *[WIP-adam]*

* Autoscaling router
  * Keep-Alive/Load-Upload (etcd) *[WIP-adam]*
  * Browser-to-peer-to-Browser (edge) *[WIP-adam]*
  * Peer-to-Peer (relay)

* Admin system
  * Database operation
  * Node monitor/manager
  * Router manager
