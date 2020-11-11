package sfu

import (
	"github.com/nats-io/nats.go"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
)

var (
	dc   string
	nid  string
	nrpc *proto.NatsRPC
	sub  *nats.Subscription
	serv *discovery.Service
)

// Init sfu
func Init(dcID string, etcdAddrs []string, natsURLs string) error {
	var err error

	dc = dcID

	if nrpc, err = proto.NewNatsRPC(natsURLs); err != nil {
		return err
	}

	if serv, err = discovery.NewService("sfu", dcID, etcdAddrs); err != nil {
		return err
	}
	nid = serv.NID()
	serv.KeepAlive()

	if sub, err = handleRequest(nid); err != nil {
		return err
	}

	return nil
}

// Close all
func Close() {
	if sub != nil {
		sub.Unsubscribe()
	}
	if nrpc != nil {
		nrpc.Close()
	}
	if serv != nil {
		serv.Close()
	}
}
