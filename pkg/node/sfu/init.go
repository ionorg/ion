package sfu

import (
	"net/http"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	isfu "github.com/pion/ion-sfu/pkg"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
)

var (
	dc   string
	nid  string
	nrpc *proto.NatsRPC
	sub  *nats.Subscription
	serv *discovery.Service
	s    *server
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

// Init sfu node
func Init(conf Config) error {
	var err error

	dc = conf.Global.Dc

	if conf.Global.Pprof != "" {
		go func() {
			log.Infof("start pprof on %s", conf.Global.Pprof)
			err := http.ListenAndServe(conf.Global.Pprof, nil)
			if err != nil {
				log.Errorf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	s = newServer(conf.Config)

	if nrpc, err = proto.NewNatsRPC(conf.Nats.URL); err != nil {
		Close()
		return err
	}

	if serv, err = discovery.NewService("sfu", dc, conf.Etcd.Addrs); err != nil {
		Close()
		return err
	}
	nid = serv.NID()
	serv.KeepAlive()

	if sub, err = handleRequest(nid); err != nil {
		Close()
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
