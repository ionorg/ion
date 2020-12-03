package islb

import (
	"net/http"
	"sync"
	"time"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
)

const (
	redisLongKeyTTL = 24 * time.Hour
)

type global struct {
	Pprof string `mapstructure:"pprof"`
	Dc    string `mapstructure:"dc"`
}

type logConf struct {
	Level string `mapstructure:"level"`
}

type etcdConf struct {
	Addrs []string `mapstructure:"addrs"`
}

type natsConf struct {
	URL string `mapstructure:"url"`
}

// Config for islb node
type Config struct {
	Global  global    `mapstructure:"global"`
	Log     logConf   `mapstructure:"log"`
	Etcd    etcdConf  `mapstructure:"etcd"`
	Nats    natsConf  `mapstructure:"nats"`
	Redis   db.Config `mapstructure:"redis"`
	CfgFile string
}

// ISLB represents islb node
type ISLB struct {
	nrpc     *proto.NatsRPC
	nodes    map[string]discovery.Node
	nodeLock sync.RWMutex
	service  *discovery.Service
	s        *server
}

// NewISLB create a islb node instance
func NewISLB() *ISLB {
	return &ISLB{
		nodes: make(map[string]discovery.Node),
	}
}

// Start islb node
func (i *ISLB) Start(conf Config) error {
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

	if i.nrpc, err = proto.NewNatsRPC(conf.Nats.URL); err != nil {
		i.Close()
		return err
	}

	if i.service, err = discovery.NewService(proto.ServiceISLB, conf.Global.Dc, conf.Etcd.Addrs); err != nil {
		i.Close()
		return err
	}
	if err := i.service.GetNodes("", i.nodes); err != nil {
		i.Close()
		return err
	}
	log.Infof("nodes up: %+v", i.nodes)
	i.service.Watch("", i.watchNodes)
	i.service.KeepAlive()

	i.s = newServer(i.service.DC(), i.service.NID(), i.nrpc, i.getNodes)
	if err = i.s.start(conf.Redis); err != nil {
		i.Close()
		return err
	}

	return nil
}

// watchNodes watch nodes
func (i *ISLB) watchNodes(state discovery.NodeState, id string, node *discovery.Node) {
	i.nodeLock.Lock()
	defer i.nodeLock.Unlock()

	if state == discovery.NodeStateUp {
		if _, found := i.nodes[id]; !found {
			i.nodes[id] = *node
		}
	} else if state == discovery.NodeStateDown {
		delete(i.nodes, id)
	}
}

func (i *ISLB) getNodes() map[string]discovery.Node {
	i.nodeLock.RLock()
	defer i.nodeLock.RUnlock()

	return i.nodes
}

// Close all
func (i *ISLB) Close() {
	if i.s != nil {
		i.s.close()
	}
	if i.service != nil {
		i.service.Close()
	}
	if i.nrpc != nil {
		i.nrpc.Close()
	}
}
