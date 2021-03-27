package avp

import (
	"fmt"
	"net/http"
	"os"
	"path"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	pb "github.com/pion/ion-avp/cmd/signal/grpc/proto"
	iavp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/elements"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/proto"
)

type global struct {
	Addr  string `mapstructure:"addr"`
	Pprof string `mapstructure:"pprof"`
	Dc    string `mapstructure:"dc"`
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
	Nats        natsConf    `mapstructure:"nats"`
	Element     elementConf `mapstructure:"element"`
	iavp.Config `mapstructure:"avp"`
}

// AVP represents avp node
type AVP struct {
	config Config
	ion.Node
	s *avpServer
}

// NewAVP create a avp node instance
func NewAVP(nid string) *AVP {
	return &AVP{Node: ion.NewNode(nid)}
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

	err = a.Node.Start(conf.Nats.URL)
	if err != nil {
		a.Close()
		return err
	}

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: proto.ServiceAVP,
		NID:     a.Node.NID,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     conf.Nats.URL,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
	}

	go a.Node.KeepAlive(node)

	elems := make(map[string]iavp.ElementFun)
	if conf.Element.Webmsaver.On {
		if _, err := os.Stat(conf.Element.Webmsaver.Path); os.IsNotExist(err) {
			if err = os.MkdirAll(conf.Element.Webmsaver.Path, 0755); err != nil {
				log.Errorf("make dir error: %v", err)
			}
		}
		elems["webmsaver"] = func(sid, pid, tid string, config []byte) iavp.Element {
			filewriter := elements.NewFileWriter(path.Join(conf.Element.Webmsaver.Path, fmt.Sprintf("%s-%s.webm", sid, pid)), 2048)
			webm := elements.NewWebmSaver()
			webm.Attach(filewriter)
			return webm
		}
	}

	a.s = newAVPServer(conf.Config, elems)
	pb.RegisterAVPServer(a.Node.ServiceRegistrar(), a.s)

	//Watch ISLB nodes.
	go a.Node.Watch(proto.ServiceISLB)

	return nil
}

// Close all
func (a *AVP) Close() {
	a.Node.Close()
}
