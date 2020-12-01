package biz

import (
	"net/http"
	"sync"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
)

var (
	dc          string // nolint: unused
	nid         string // nolint: unused
	subs        map[string]*nats.Subscription
	avpElements []string
	nrpc        *proto.NatsRPC
	nodeLock    sync.RWMutex
	nodes       map[string]discovery.Node
	serv        *discovery.Service
	signal      *server
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

type avp struct {
	Elements []string `mapstructure:"elements"`
}

// Config for biz node
type Config struct {
	Global global     `mapstructure:"global"`
	Log    logConf    `mapstructure:"log"`
	Etcd   etcdConf   `mapstructure:"etcd"`
	Nats   natsConf   `mapstructure:"nats"`
	Signal signalConf `mapstructure:"signal"`
	Avp    avp        `mapstructure:"avp"`
}

// Init biz node
func Init(conf Config) error {
	var err error

	dc = conf.Global.Dc
	avpElements = conf.Avp.Elements
	nodes = make(map[string]discovery.Node)
	subs = make(map[string]*nats.Subscription)

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

	if serv, err = discovery.NewService("biz", conf.Global.Dc, conf.Etcd.Addrs); err != nil {
		Close()
		return err
	}
	if err := serv.GetNodes("islb", nodes); err != nil {
		Close()
		return err
	}
	log.Infof("nodes up: %+v", nodes)
	nid = serv.NID()
	for _, n := range nodes {
		if n.Service == "islb" {
			subIslbBroadcast(&n)
		}
	}
	serv.Watch("islb", watchIslbNodes)
	serv.KeepAlive()

	signal = newServer(conf.Signal)

	return nil
}

// Close all
func Close() {
	closeSubs()
	if signal != nil {
		signal.close()
	}
	if nrpc != nil {
		nrpc.Close()
	}
	if serv != nil {
		serv.Close()
	}
}

func subIslbBroadcast(node *discovery.Node) {
	log.Infof("subscribe islb broadcast: %s", node.NID)
	if sub, err := nrpc.Subscribe(node.NID+"-event", handleIslbBroadcast); err == nil {
		subs[node.ID()] = sub
	} else {
		log.Errorf("subcribe error: %v", err)
	}
}

// watchNodes watch islb nodes up/down
func watchIslbNodes(state discovery.State, id string, node *discovery.Node) {
	nodeLock.Lock()
	defer nodeLock.Unlock()

	if state == discovery.NodeUp {
		if _, found := nodes[id]; !found {
			nodes[id] = *node
		}
		if _, found := subs[id]; !found {
			subIslbBroadcast(node)
		}
	} else if state == discovery.NodeDown {
		if sub := subs[id]; sub != nil {
			sub.Unsubscribe()
		}
		delete(subs, id)
		delete(nodes, id)
	}
}

func getNodes() map[string]discovery.Node {
	nodeLock.RLock()
	defer nodeLock.RUnlock()

	return nodes
}

func closeSubs() {
	nodeLock.Lock()
	defer nodeLock.Unlock()

	for id, s := range subs {
		if err := s.Unsubscribe(); err != nil {
			log.Errorf("unsubscribe %s error: %v", id, err)
		}
	}
}
