package biz

import (
	"net/http"
	"sync"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
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

type avpConf struct {
	Elements []string `mapstructure:"elements"`
}

// Config for biz node
type Config struct {
	Global global   `mapstructure:"global"`
	Log    logConf  `mapstructure:"log"`
	Etcd   etcdConf `mapstructure:"etcd"`
	Nats   natsConf `mapstructure:"nats"`
	Avp    avpConf  `mapstructure:"avp"`
}

// BIZ represents biz node
type BIZ struct {
	conf     Config
	nrpc     *proto.NatsRPC
	sub      *nats.Subscription
	subs     map[string]*nats.Subscription
	nodeLock sync.RWMutex
	nodes    map[string]discovery.Node
	service  *discovery.Service
	s        *Server
}

// NewBIZ create a biz node instance
func NewBIZ(conf Config) *BIZ {
	return &BIZ{
		conf:  conf,
		nodes: make(map[string]discovery.Node),
		subs:  make(map[string]*nats.Subscription),
	}
}

// Start biz node
func (b *BIZ) Start() (*Server, error) {
	var err error

	if b.conf.Global.Pprof != "" {
		go func() {
			log.Infof("start pprof on %s", b.conf.Global.Pprof)
			err := http.ListenAndServe(b.conf.Global.Pprof, nil)
			if err != nil {
				log.Errorf("http.ListenAndServe err=%v", err)
			}
		}()
	}

	if b.nrpc, err = proto.NewNatsRPC(b.conf.Nats.URL); err != nil {
		b.Close()
		return nil, err
	}

	if b.service, err = discovery.NewService(proto.ServiceBIZ, b.conf.Global.Dc, b.conf.Etcd.Addrs); err != nil {
		b.Close()
		return nil, err
	}
	if err = b.service.GetNodes(proto.ServiceISLB, b.nodes); err != nil {
		b.Close()
		return nil, err
	}
	log.Infof("nodes up: %+v", b.nodes)
	for _, n := range b.nodes {
		if n.Service == proto.ServiceISLB {
			b.subIslbBroadcast(n)
		}
	}
	b.service.Watch(proto.ServiceISLB, b.watchIslbNodes)
	b.service.KeepAlive()

	b.s = newServer(b.conf.Global.Dc, b.service.NID(), b.conf.Avp.Elements, b.nrpc, b.getNodes)
	if err = b.s.start(); err != nil {
		return nil, err
	}

	return b.s, nil
}

// Close all
func (b *BIZ) Close() {
	b.closeSubs()
	if b.s != nil {
		b.s.close()
	}
	if b.sub != nil {
		if err := b.sub.Unsubscribe(); err != nil {
			log.Errorf("unsubscribe %s error: %v", b.sub.Subject, err)
		}
	}
	if b.service != nil {
		b.service.Close()
	}
	if b.nrpc != nil {
		b.nrpc.Close()
	}
}

func (b *BIZ) subIslbBroadcast(node discovery.Node) {
	log.Infof("subscribe islb broadcast: %s", node.NID)
	if sub, err := b.nrpc.Subscribe(node.NID+"-event", b.handleIslbBroadcast); err == nil {
		b.subs[node.ID()] = sub
	} else {
		log.Errorf("subcribe error: %v", err)
	}
}

func (b *BIZ) handleIslbBroadcast(msg interface{}) (interface{}, error) {
	return b.s.broadcast(msg)
}

// watchNodes watch islb nodes up/down
func (b *BIZ) watchIslbNodes(state discovery.NodeState, id string, node *discovery.Node) {
	b.nodeLock.Lock()
	defer b.nodeLock.Unlock()

	if state == discovery.NodeStateUp {
		if _, found := b.nodes[id]; !found {
			b.nodes[id] = *node
		}
		if _, found := b.subs[id]; !found {
			b.subIslbBroadcast(*node)
		}
	} else if state == discovery.NodeStateDown {
		if sub := b.subs[id]; sub != nil {
			if err := sub.Unsubscribe(); err != nil {
				log.Errorf("unsubscribe %s error: %v", sub.Subject, err)
			}
		}
		delete(b.subs, id)
		delete(b.nodes, id)
	}
}

func (b *BIZ) getNodes() map[string]discovery.Node {
	b.nodeLock.RLock()
	defer b.nodeLock.RUnlock()

	return b.nodes
}

func (b *BIZ) closeSubs() {
	b.nodeLock.Lock()
	defer b.nodeLock.Unlock()

	for _, sub := range b.subs {
		if err := sub.Unsubscribe(); err != nil {
			log.Errorf("unsubscribe %s error: %v", sub.Subject, err)
		}
	}
}
