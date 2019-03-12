# SFU

## 1. Architecture

![arch](arch.png)

## 2. Roadmap

* Media stream support
  * Peer-to-Browser (webrtc)
  * Peer-to-Peer (rtp)

* Singal exchange support
  * Singal protocol (room/stream/relay )
  * Exchange sdp/candidate/other (edge)

* Autoscaling router
  * Keep-Alive/Load-Upload (etcd)
  * Browser-to-peer-to-Browser (edge)
  * Peer-to-Peer (relay)

* Admin system
  * Database operation
  * Node monitor/manager
  * Router manager
