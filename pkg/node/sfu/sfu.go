package sfu

import (
	"net/http"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	isfu "github.com/pion/ion-sfu/pkg/sfu"
	psfu "github.com/pion/ion/pkg/grpc/sfu"
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
		Node: ion.Node{NID: nid},
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
			Addr:     s.Node.NID,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
	}

	go s.Node.KeepAlive(node)

	s.s = newServer(isfu.NewSFU(conf.Config))
	//grpc service
	psfu.RegisterSFUServer(s.Node.ServiceRegistrar(), s.s)

	//Watch ISLB nodes.
	go s.Node.Watch(proto.ServiceISLB, s.s.watchIslbNodes)
	return nil
}

// Close all
func (s *SFU) Close() {
	s.Node.Close()
}
