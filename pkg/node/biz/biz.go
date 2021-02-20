package biz

import (
	"net/http"
	"sync"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/pkg/grpc/biz"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/proto"
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

type avpConf struct {
	Elements []string `mapstructure:"elements"`
}

// Config for biz node
type Config struct {
	Global global   `mapstructure:"global"`
	Log    logConf  `mapstructure:"log"`
	Nats   natsConf `mapstructure:"nats"`
	Avp    avpConf  `mapstructure:"avp"`
}

// BIZ represents biz node
type BIZ struct {
	ion.Node
	conf     Config
	nodeLock sync.RWMutex
	s        *Server
}

// NewBIZ create a biz node instance
func NewBIZ(nid string) *BIZ {
	return &BIZ{
		Node: ion.Node{NID: nid},
	}
}

// Start biz node
func (b *BIZ) Start(conf Config) (*Server, error) {
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

	err = b.Node.Start(conf.Nats.URL)
	if err != nil {
		b.Close()
		return nil, err
	}

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: proto.ServiceBIZ,
		NID:     b.Node.NID,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     b.Node.NID,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
	}

	go b.Node.KeepAlive(node)

	//b.netcd.Watch(proto.ServiceISLB, func(tate discovery.NodeState, node *discovery.Node) {
	//
	//})
	b.s = &Server{
		elements: conf.Avp.Elements,
	}
	pb.RegisterBizServer(b.Node.ServiceRegistrar(), b.s)
	return b.s, nil
}

// Close all
func (b *BIZ) Close() {
	b.Node.Close()
}

/*
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
*/
