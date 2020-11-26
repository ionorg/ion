package islb

import (
	"net/http"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
)

const (
	redisLongKeyTTL = 24 * time.Hour
)

var (
	dc       string
	nid      string
	bid      string
	nrpc     *proto.NatsRPC
	sub      *nats.Subscription
	redis    *db.Redis
	nodes    map[string]discovery.Node
	nodeLock sync.RWMutex
	serv     *discovery.Service
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

// Init islb
func Init(conf Config) error {
	var err error

	dc = conf.Global.Dc
	nodes = make(map[string]discovery.Node)

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

	redis = db.NewRedis(conf.Redis)

	if serv, err = discovery.NewService("islb", dc, conf.Etcd.Addrs); err != nil {
		Close()
		return err
	}
	if err := serv.GetNodes("", nodes); err != nil {
		Close()
		return err
	}
	log.Infof("nodes up: %+v", nodes)
	nid = serv.NID()
	bid = nid + "-event"
	serv.Watch("", watchNodes)
	serv.KeepAlive()

	if sub, err = handleRequest(nid); err != nil {
		return err
	}

	return nil
}

// watchNodes watch nodes
func watchNodes(state discovery.State, id string, node *discovery.Node) {
	nodeLock.Lock()
	defer nodeLock.Unlock()

	if state == discovery.NodeUp {
		if _, found := nodes[id]; !found {
			nodes[id] = *node
		}
	} else if state == discovery.NodeDown {
		if _, found := nodes[id]; found {
			delete(nodes, id)
		}
	}
}

func getNodes() map[string]discovery.Node {
	nodeLock.RLock()
	defer nodeLock.RUnlock()

	return nodes
}

// Close all
func Close() {
	if sub != nil {
		sub.Unsubscribe()
	}
	if nrpc != nil {
		nrpc.Close()
	}
	if redis != nil {
		redis.Close()
	}
	if serv != nil {
		serv.Close()
	}
}
