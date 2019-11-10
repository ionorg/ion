package signal

import (
	"sync"

	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/server"
)

type AcceptFunc peer.AcceptFunc
type RejectFunc peer.RejectFunc

const (
	errInvalidMethod = "method not found"
	errInvalidData   = "data not found"
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
}
