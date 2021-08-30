package avp

import (
	"fmt"
	"os"
	"path"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	pb "github.com/pion/ion-avp/cmd/signal/grpc/proto"
	iavp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/elements"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
)

type global struct {
	Addr string `mapstructure:"addr"`
	Dc   string `mapstructure:"dc"`
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
	ion.Node
	s *avpServer
}

// NewAVP create a avp node instance
func NewAVP() *AVP {
	return &AVP{Node: ion.NewNode("avp-" + util.RandomString(6))}
}

// Start avp node
func (a *AVP) Start(conf Config) error {
	var err error

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

	go func() {
		err := a.Node.KeepAlive(node)
		if err != nil {
			log.Errorf("avp.Node.KeepAlive: error => %v", err)
		}
	}()

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

	//Watch ALL nodes.
	go func() {
		err := a.Node.Watch(proto.ServiceALL)
		if err != nil {
			log.Errorf("Node.Watch(proto.ServiceALL) error %v", err)
		}
	}()

	return nil
}

// Close all
func (a *AVP) Close() {
	a.Node.Close()
}
