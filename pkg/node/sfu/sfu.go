package sfu

import (
	"net/http"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion-sfu/cmd/signal/grpc/proto"
	"github.com/pion/ion-sfu/pkg/middlewares/datachannel"
	isfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/proto"
)

type global struct {
	Pprof string `mapstructure:"pprof"`
	Dc    string `mapstructure:"dc"`
}

type natsConf struct {
	URL string `mapstructure:"url"`
}

// Config for sfu node
type Config struct {
	Global global   `mapstructure:"global"`
	Nats   natsConf `mapstructure:"nats"`
	isfu.Config
}

// SFU represents a sfu node
type SFU struct {
	ion.Node
	s *sfuServer
}

// NewSFU create a sfu node instance
func NewSFU(nid string) *SFU {
	s := &SFU{
		Node: ion.NewNode(nid),
	}
	return s
}

// Start sfu node
func (s *SFU) Start(conf Config) error {
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

	err = s.Node.Start(conf.Nats.URL)
	if err != nil {
		s.Close()
		return err
	}

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: proto.ServiceSFU,
		NID:     s.Node.NID,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     conf.Nats.URL,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
	}

	go s.Node.KeepAlive(node)

	nsfu := isfu.NewSFU(conf.Config)
	dc := nsfu.NewDatachannel(isfu.APIChannelLabel)
	dc.Use(datachannel.SubscriberAPI)

	s.s = newSFUServer(s, nsfu, s.NatsConn())
	//grpc service
	pb.RegisterSFUServer(s.Node.ServiceRegistrar(), s.s)

	//Watch ISLB nodes.
	go s.Node.Watch(proto.ServiceISLB)
	return nil
}

// Close all
func (s *SFU) Close() {
	s.Node.Close()
}
