package avp

import (
	"fmt"
	"net/http"
	"os"
	"path"

	client "github.com/cloudwebrtc/nats-discovery/pkg/client"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/nats-io/nats.go"
	iavp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/elements"
	log "github.com/pion/ion-log"
	proto "github.com/pion/ion/pkg/grpc/avp"
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
	s     *avpServer
	nc    *nats.Conn
	ngrpc *rpc.Server
	netcd *client.Client
	nid   string
}

// NewAVP create a avp node instance
func NewAVP(nid string) *AVP {
	return &AVP{nid: nid}
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

	// connect options
	opts := []nats.Option{nats.Name("nats sfu service")}
	//opts = setupConnOptions(opts)

	// connect to nats server
	if a.nc, err = nats.Connect(conf.Nats.URL, opts...); err != nil {
		a.Close()
		return err
	}

	a.netcd, err = client.NewClient(a.nc)

	if err != nil {
		a.Close()
		return err
	}

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: "avp",
		NID:     a.nid,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     a.nid,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
	}

	go a.netcd.KeepAlive(node)

	a.s = &avpServer{}
	//grpc service
	a.ngrpc = rpc.NewServer(a.nc, a.nid)
	proto.RegisterAVPServer(a.ngrpc, a.s)

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

	return nil
}

// Close all
func (a *AVP) Close() {
	if a.ngrpc != nil {
		a.ngrpc.Stop()
	}
	if a.nc != nil {
		a.nc.Close()
	}
}
