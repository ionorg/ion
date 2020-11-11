package avp

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

// Init avp
func Init(dcID string, etcdAddrs []string, natsURLs string, avpConf *Config) error {
	dc = dcID

	var err error

	if nrpc, err = proto.NewNatsRPC(natsURLs); err != nil {
		return err
	}

	if serv, err = discovery.NewService("avp", dcID, etcdAddrs); err != nil {
		return err
	}
	nid = serv.NID()
	serv.KeepAlive()

	if sub, err = handleRequest(nid); err != nil {
		return err
	}

	initAVP(avpConf)

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
