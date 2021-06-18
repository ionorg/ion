package server

import (
	"net/http"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/apps/biz/proto"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/proto"
)

type global struct {
	Pprof string `mapstructure:"pprof"`
	Dc    string `mapstructure:"dc"`
}

type logConf struct {
	Level string `mapstructure:"level"`
}

type natsConf struct {
	URL string `mapstructure:"url"`
}

type nodeConf struct {
	NID string `mapstructure:"nid"`
}

// Config for biz node
type Config struct {
	Global global   `mapstructure:"global"`
	Log    logConf  `mapstructure:"log"`
	Nats   natsConf `mapstructure:"nats"`
	Node   nodeConf `mapstructure:"node"`
}

// BIZ represents biz node
type BIZ struct {
	ion.Node
	s *BizServer
}

// NewBIZ create a biz node instance
func NewBIZ(nid string) *BIZ {
	return &BIZ{
		Node: ion.NewNode(nid),
	}
}

// Start biz node
func (b *BIZ) Start(conf Config) error {
	var err error

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("start pprof on %s", conf.Global.Pprof)
			err := http.ListenAndServe(conf.Global.Pprof, nil)
			if err != nil {
				log.Errorf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	err = b.Node.Start(conf.Nats.URL)
	if err != nil {
		b.Close()
		return err
	}

	b.s, err = newBizServer(b, conf.Global.Dc, b.NID, b.NatsConn())
	if err != nil {
		b.Close()
		return err
	}

	pb.RegisterBizServer(b.Node.ServiceRegistrar(), b.s)

	// Register reflection service on nats-rpc server.
	reflection.Register(b.Node.ServiceRegistrar().(*nrpc.Server))

	go b.s.stat()

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: proto.ServiceBIZ,
		NID:     b.Node.NID,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     conf.Nats.URL,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
	}

	go func() {
		err := b.Node.KeepAlive(node)
		if err != nil {
			log.Errorf("biz.Node.KeepAlive(%v) error %v", b.Node.NID, err)
		}
	}()

	//Watch ALL nodes.
	go func() {
		err := b.Node.Watch(proto.ServiceALL)
		if err != nil {
			log.Errorf("Node.Watch(proto.ServiceALL) error %v", err)
		}
	}()
	return nil
}

// Close all
func (b *BIZ) Close() {
	b.s.close()
	b.Node.Close()
}

// Service return grpc services.
func (b *BIZ) Service() *BizServer {
	return b.s
}
