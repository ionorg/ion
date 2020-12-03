package sfu

import (
	"net/http"

	log "github.com/pion/ion-log"
	isfu "github.com/pion/ion-sfu/pkg"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
)

type global struct {
	Pprof string `mapstructure:"pprof"`
	Dc    string `mapstructure:"dc"`
}

type etcdConf struct {
	Addrs []string `mapstructure:"addrs"`
}

type natsConf struct {
	URL string `mapstructure:"url"`
}

// Config for sfu node
type Config struct {
	Global global   `mapstructure:"global"`
	Etcd   etcdConf `mapstructure:"etcd"`
	Nats   natsConf `mapstructure:"nats"`
	isfu.Config
}

// SFU represents a sfu node
type SFU struct {
	nrpc    *proto.NatsRPC
	service *discovery.Service
	s       *server
}

// NewSFU create a sfu node instance
func NewSFU() *SFU {
	return &SFU{}
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

	if s.nrpc, err = proto.NewNatsRPC(conf.Nats.URL); err != nil {
		s.Close()
		return err
	}

	if s.service, err = discovery.NewService(proto.ServiceSFU, conf.Global.Dc, conf.Etcd.Addrs); err != nil {
		s.Close()
		return err
	}
	s.service.KeepAlive()

	s.s = newServer(conf.Config, s.service.NID(), s.nrpc)
	if err := s.s.start(); err != nil {
		return err
	}

	return nil
}

// Close all
func (s *SFU) Close() {
	if s.s != nil {
		s.s.close()
	}
	if s.nrpc != nil {
		s.nrpc.Close()
	}
	if s.service != nil {
		s.service.Close()
	}
}
