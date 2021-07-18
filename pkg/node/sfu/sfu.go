package sfu

import (
	"fmt"
	"net/http"
	"os"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	log "github.com/pion/ion-log"
	"github.com/pion/ion-sfu/pkg/middlewares/datachannel"
	isfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/runner"
	"github.com/pion/ion/pkg/util"
	pb "github.com/pion/ion/proto/sfu"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

const (
	portRangeLimit = 100
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

// Config for sfu node
type Config struct {
	runner.ConfigBase
	Global global   `mapstructure:"global"`
	Log    logConf  `mapstructure:"log"`
	Nats   natsConf `mapstructure:"nats"`
	isfu.Config
}

func unmarshal(rawVal interface{}) error {
	if err := viper.Unmarshal(rawVal); err != nil {
		return err
	}
	return nil
}

func (c *Config) Load(file string) error {
	_, err := os.Stat(file)
	if err != nil {
		return err
	}

	viper.SetConfigFile(file)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		log.Errorf("config file %s read failed. %v\n", file, err)
		return err
	}

	err = unmarshal(c)
	if err != nil {
		return err
	}
	err = unmarshal(&c.Config)
	if err != nil {
		return err
	}
	if err != nil {
		log.Errorf("config file %s loaded failed. %v\n", file, err)
		return err
	}

	if len(c.WebRTC.ICEPortRange) > 2 {
		err = fmt.Errorf("config file %s loaded failed. range port must be [min,max]", file)
		log.Errorf("err=%v", err)
		return err
	}

	if len(c.WebRTC.ICEPortRange) != 0 && c.WebRTC.ICEPortRange[1]-c.WebRTC.ICEPortRange[0] < portRangeLimit {
		err = fmt.Errorf("config file %s loaded failed. range port must be [min, max] and max - min >= %d", file, portRangeLimit)
		log.Errorf("err=%v", err)
		return err
	}

	log.Infof("config %s load ok!\n", file)
	return nil
}

// SFU represents a sfu node
type SFU struct {
	runner.Service
	ion.Node
	s    *sfuServer
	conf Config
}

// New create a sfu node instance
func New(conf Config) *SFU {
	s := &SFU{
		conf: conf,
		Node: ion.NewNode("sfu-" + util.RandomString(4)),
	}
	return s
}

func (s *SFU) ConfigBase() runner.ConfigBase {
	return &s.conf
}

// StartGRPC start with grpc.ServiceRegistrar
func (s *SFU) StartGRPC(registrar grpc.ServiceRegistrar) error {
	if s.conf.Global.Pprof != "" {
		go func() {
			log.Infof("start pprof on %s", s.conf.Global.Pprof)
			err := http.ListenAndServe(s.conf.Global.Pprof, nil)
			if err != nil {
				log.Warnf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	s.s = newSFUServer(s, isfu.NewSFU(s.conf.Config))
	pb.RegisterSFUServer(registrar, s.s)

	return nil
}

// Start sfu node
func (s *SFU) Start() error {
	var err error

	if s.conf.Global.Pprof != "" {
		go func() {
			log.Infof("start pprof on %s", s.conf.Global.Pprof)
			err := http.ListenAndServe(s.conf.Global.Pprof, nil)
			if err != nil {
				log.Errorf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	err = s.Node.Start(s.conf.Nats.URL)
	if err != nil {
		s.Close()
		return err
	}

	nsfu := isfu.NewSFU(s.conf.Config)
	dc := nsfu.NewDatachannel(isfu.APIChannelLabel)
	dc.Use(datachannel.SubscriberAPI)

	s.s = newSFUServer(s, nsfu)
	//grpc service
	pb.RegisterSFUServer(s.Node.ServiceRegistrar(), s.s)

	// Register reflection service on nats-rpc server.
	reflection.Register(s.Node.ServiceRegistrar().(*nrpc.Server))

	node := discovery.Node{
		DC:      s.conf.Global.Dc,
		Service: proto.ServiceSFU,
		NID:     s.Node.NID,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     s.conf.Nats.URL,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
	}

	go func() {
		err := s.Node.KeepAlive(node)
		if err != nil {
			log.Errorf("sfu.Node.KeepAlive(%v) error %v", s.Node.NID, err)
		}
	}()

	//Watch ALL nodes.
	go func() {
		err := s.Node.Watch(proto.ServiceALL)
		if err != nil {
			log.Errorf("Node.Watch(proto.ServiceALL) error %v", err)
		}
	}()

	return nil
}

// Close all
func (s *SFU) Close() {
	s.Node.Close()
}
