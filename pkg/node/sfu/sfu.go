package sfu

import (
	"net/http"

	client "github.com/cloudwebrtc/nats-discovery/pkg/client"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	isfu "github.com/pion/ion-sfu/pkg"
	proto "github.com/pion/ion/pkg/grpc/rtc"
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
	s     *sfuServer
	nc    *nats.Conn
	ngrpc *rpc.Server
	netcd *client.Client
	nid   string
}

// NewSFU create a sfu node instance
func NewSFU(nid string) *SFU {
	return &SFU{nid: nid}
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
	// connect options
	opts := []nats.Option{nats.Name("nats sfu service")}
	//opts = setupConnOptions(opts)

	// connect to nats server
	if s.nc, err = nats.Connect(conf.Nats.URL, opts...); err != nil {
		s.Close()
		return err
	}

	s.netcd, err = client.NewClient(s.nc)

	if err != nil {
		s.Close()
		return err
	}

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: "sfu",
		NID:     s.nid,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     s.nid,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
	}

	go s.netcd.KeepAlive(node)

	s.s = newServer(isfu.NewSFU(conf.Config))
	//grpc service
	s.ngrpc = rpc.NewServer(s.nc, s.nid)
	proto.RegisterRTCServer(s.ngrpc, s.s)

	return nil
}

// Close all
func (s *SFU) Close() {
	if s.ngrpc != nil {
		s.ngrpc.Stop()
	}
	if s.nc != nil {
		s.nc.Close()
	}
}
