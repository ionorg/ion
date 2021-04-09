
## Distributed System
  - [x] Service Registration and Discovery - ETCD
  - [x] Auto Scale
    - [x] Docker
    - [x] K8S - *TODO*
  - [x] MQ - NATS
    - [x] RPC
    - [x] Broadcast

## Decoupling Nodes
  - [x] BIZ - Signal and business logic server 
  - [x] ISLB - Dispatch and control server
  - [x] SFU - Selective Forward Unit
  - [x] AVP - Audio and video process server

## Popular Media Stack

  - [x] SFU
    - [x] Router
    - [x] Plugin
      - [x] JitterBuffer
        - [x] Nack
        - [x] PLI
        - [x] Lite-REMB
        - [x] Transport-CC - *experiment*
    - [x] Transport
      - [x] WebRTCTransport
      - [x] RTPTransport (over KCP)
  - [x] AVP
    - [x] Record Webm
    - [x] OpenCV - *WIP*

## High Performance

  - [x] Live Mode: 1-3 Publisher: 1000+ Subscribers [StressTest](production/stress_test.md)
  - [x] Communication mode: 50 : 50 - *TODO*
  - [x] Low latency < 500ms 

## Auto scale and Loadbalance
  
  - [x] Auto scale - *WIP*
  - [x] Auto banlance - *WIP*
  - [x] Relay logic - *WIP*

