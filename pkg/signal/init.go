package signal

import (
	"fmt"
	"sync"
	"time"

	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/server"
	"github.com/pion/ion/pkg/log"
)

type AcceptFunc peer.AcceptFunc
type RejectFunc peer.RejectFunc

const (
	errInvalidMethod = "method not found"
	errInvalidData   = "data not found"

	statCycle = time.Second * 3
)

var (
	bizCall  func(method string, peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc)
	wsServer *server.WebSocketServer
	rooms    = make(map[string]*Room)
	roomLock sync.RWMutex
)

func Init(host string, port int, cert, key string, bizEntry interface{}) {
	wsServer = server.NewWebSocketServer(in)
	config := server.DefaultConfig()
	config.Host = host
	config.Port = port
	config.CertFile = cert
	config.KeyFile = key
	bizCall = bizEntry.(func(method string, peer *Peer, msg map[string]interface{}, accept AcceptFunc, reject RejectFunc))
	go wsServer.Bind(config)
	go stat()
}

func stat() {
	t := time.NewTicker(statCycle)
	defer t.Stop()
	for range t.C {
		info := "\n----------------signal-----------------\n"
		roomLock.Lock()
		for rid, room := range rooms {
			info += fmt.Sprintf("room: %s\npeers: %d\n", rid, len(room.GetPeers()))
			if len(room.GetPeers()) == 0 {
				delete(rooms, rid)
			}
		}
		roomLock.Unlock()
		log.Infof(info)
	}
}
