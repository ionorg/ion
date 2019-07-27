package signal

import (
	"github.com/cloudwebrtc/go-protoo/server"
	"github.com/pion/ion/conf"
)

var (
	wsServer *server.WebSocketServer
)

func Start() {
	wsServer = server.NewWebSocketServer(in)
	config := server.DefaultConfig()
	config.Host = conf.Signal.Host
	config.Port = conf.Signal.Port
	config.CertFile = conf.Signal.Cert
	config.KeyFile = conf.Signal.Key
	go wsServer.Bind(config)
}
