package avp

import (
	"fmt"
	"net/http"
	"os"
	"path"

	iavp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/elements"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
)

type global struct {
	Addr  string `mapstructure:"addr"`
	Pprof string `mapstructure:"pprof"`
	Dc    string `mapstructure:"dc"`
}

type etcdConf struct {
	Addrs []string `mapstructure:"addrs"`
}

type natsConf struct {
	URL string `mapstructure:"url"`
}

type webmsaver struct {
	On   bool   `mapstructure:"on"`
	Path string `mapstructure:"path"`
}
type elementConf struct {
	Webmsaver webmsaver `mapstructure:"webmsaver"`
}

// Config for avp node
type Config struct {
	Global      global      `mapstructure:"global"`
	Etcd        etcdConf    `mapstructure:"etcd"`
	Nats        natsConf    `mapstructure:"nats"`
	Element     elementConf `mapstructure:"element"`
	iavp.Config `mapstructure:"avp"`
}

// AVP represents avp node
type AVP struct {
	nrpc    *proto.NatsRPC
	service *discovery.Service
	s       *server
}

// NewAVP create a avp node instance
func NewAVP() *AVP {
	return &AVP{}
}

// Start avp node
func (a *AVP) Start(conf Config) error {
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

	if a.nrpc, err = proto.NewNatsRPC(conf.Nats.URL); err != nil {
		a.Close()
		return err
	}

	if a.service, err = discovery.NewService("avp", conf.Global.Dc, conf.Etcd.Addrs); err != nil {
		a.Close()
		return err
	}
	a.service.KeepAlive()

	elems := make(map[string]iavp.ElementFun)
	if conf.Element.Webmsaver.On {
		if _, err := os.Stat(conf.Element.Webmsaver.Path); os.IsNotExist(err) {
			if err = os.MkdirAll(conf.Element.Webmsaver.Path, 0755); err != nil {
				log.Errorf("make dir error: %v", err)
			}
		}
		elems["webmsaver"] = func(sid, pid, tid string, config []byte) iavp.Element {
			filewriter := elements.NewFileWriter(path.Join(conf.Element.Webmsaver.Path, fmt.Sprintf("%s-%s.webm", sid, pid)))
			webm := elements.NewWebmSaver()
			webm.Attach(filewriter)
			return webm
		}
	}
	a.s = newServer(conf.Config, elems, a.service.NID(), a.nrpc)
	if err = a.s.start(); err != nil {
		a.Close()
		return err
	}

	return nil
}

// Close all
func (a *AVP) Close() {
	if a.s != nil {
		a.s.close()
	}
	if a.nrpc != nil {
		a.nrpc.Close()
	}
	if a.service != nil {
		a.service.Close()
	}
}
