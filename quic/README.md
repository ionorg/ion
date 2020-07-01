<h1 align="center">
  <br>
  Pion QUIC
  <br>
</h1>
<h4 align="center">A Go implementation of the QUIC WebRTC & CS API</h4>
<p align="center">
  <a href="https://pion.ly"><img src="https://img.shields.io/badge/pion-webrtc-gray.svg?longCache=true&colorB=brightgreen" alt="Pion quic"></a>
  <!--<a href="https://sourcegraph.com/github.com/sssgun/ion/webrtc?badge"><img src="https://sourcegraph.com/github.com/sssgun/ion/webrtc/-/badge.svg" alt="Sourcegraph Widget"></a>-->
  <a href="https://pion.ly/slack"><img src="https://img.shields.io/badge/join-us%20on%20slack-gray.svg?longCache=true&logo=slack&colorB=brightgreen" alt="Slack Widget"></a>
  <a href="https://waffle.io/pion/webrtc"><img src="https://img.shields.io/badge/pm-waffle-gray.svg?longCache=true&colorB=brightgreen" alt="Waffle board"></a>
  <br>
  <a href="https://travis-ci.org/pion/quic"><img src="https://travis-ci.org/pion/quic.svg?branch=master" alt="Build Status"></a>
  <a href="https://godoc.org/github.com/sssgun/ion/quic"><img src="https://godoc.org/github.com/sssgun/ion/quic?status.svg" alt="GoDoc"></a>
  <a href="https://coveralls.io/github/pion/quic"><img src="https://coveralls.io/repos/github/pion/quic/badge.svg" alt="Coverage Status"></a>
  <a href="https://goreportcard.com/report/github.com/sssgun/ion/quic"><img src="https://goreportcard.com/badge/github.com/sssgun/ion/quic" alt="Go Report Card"></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
</p>
<br>

pion/quic implements the following experimental QUIC APIs:
- [QUIC API for Peer-to-peer Connections](https://w3c.github.io/webrtc-quic/)
- [QUIC API for Client-to-Server Connections](https://w3c.github.io/webrtc-quic/cs)

The library doesn't implement the QUIC protocol itself. It relies on [quic-go](https://github.com/lucas-clemente/quic-go) for this purpose.

### Usage
The Pion QUIC API aims to closely match the JavaScript API specs. Most existing documentation should therefore be useful when working with Pion. Furthermore, our **[GoDoc](https://godoc.org/github.com/sssgun/ion/quic)** is actively maintained.

### Roadmap
The library is used as a part of our WebRTC implementation. Please refer to that [roadmap](https://github.com/sssgun/ion/webrtc/issues/9) to track our major milestones.

### Community
Pion has an active community on the [Golang Slack](https://invite.slack.golangbridge.org/). Sign up and join the **#pion** channel for discussions and support. You can also use [Pion mailing list](https://groups.google.com/forum/#!forum/pion).

We are always looking to support **your projects**. Please reach out if you have something to build!

If you need commercial support or don't want to use public methods you can contact us at [team@pion.ly](mailto:team@pion.ly)

### Contributing
Check out the **[contributing wiki](https://github.com/sssgun/ion/webrtc/wiki/Contributing)** to join the group of amazing people making this project possible:

* [Sean DuBois](https://github.com/Sean-Der) - *Original Author*
* [Michiel De Backker](https://github.com/backkem) - *Original Author*
* [Max Hawkins](https://github.com/maxhawkins) - *Cleanup RTC prefix*

### License
MIT License - see [LICENSE](LICENSE) for full text
