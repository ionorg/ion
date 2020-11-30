package avp

import (
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/nats-io/nats.go"
	iavp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/elements"
	log "github.com/pion/ion-log"
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

// Init avp node
func Init(conf Config) error {
	dc = conf.Global.Dc

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

	if nrpc, err = proto.NewNatsRPC(conf.Nats.URL); err != nil {
		Close()
		return err
	}

	if serv, err = discovery.NewService("avp", dc, conf.Etcd.Addrs); err != nil {
		Close()
		return err
	}
	nid = serv.NID()
	serv.KeepAlive()

	if sub, err = handleRequest(nid); err != nil {
		Close()
		return err
	}

	elems := make(map[string]iavp.ElementFun)
	if conf.Element.Webmsaver.On {
		if _, err := os.Stat(conf.Element.Webmsaver.Path); os.IsNotExist(err) {
			if err = os.MkdirAll(conf.Element.Webmsaver.Path, 0755); err != nil {
				log.Errorf("make dir error: %v", err)
			}
		}
		elems["webmsaver"] = func(rid, pid, tid string, config []byte) iavp.Element {
			filewriter := elements.NewFileWriter(path.Join(conf.Element.Webmsaver.Path, fmt.Sprintf("%s-%s.webm", rid, pid)))
			webm := elements.NewWebmSaver()
			webm.Attach(filewriter)
			return webm
		}
	}
	s = newServer(conf.Config, elems)

	return nil
}

// Close all
func Close() {
	if sub != nil {
		if err := sub.Unsubscribe(); err != nil {
			log.Errorf("unsubscribe %s error: %v", nid, err)
		}
	}
	if nrpc != nil {
		nrpc.Close()
	}
	if serv != nil {
		serv.Close()
	}
}
