package islb

import (
	"errors"
	"net/http"
	"time"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/db"
	pb "github.com/pion/ion/pkg/grpc/islb"
	"github.com/pion/ion/pkg/ion"
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

type natsConf struct {
	URL string `mapstructure:"url"`
}

// Config for islb node
type Config struct {
	Global  global    `mapstructure:"global"`
	Log     logConf   `mapstructure:"log"`
	Nats    natsConf  `mapstructure:"nats"`
	Redis   db.Config `mapstructure:"redis"`
	CfgFile string
}

// ISLB represents islb node
type ISLB struct {
	ion.Node
	s        *islbServer
	registry *discovery.Registry
	redis    *db.Redis
}

// NewISLB create a islb node instance
func NewISLB(nid string) *ISLB {
	return &ISLB{Node: ion.NewNode(nid)}
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

	err = i.Node.Start(conf.Nats.URL)
	if err != nil {
		i.Close()
		return err
	}

	//registry for node discovery.
	i.registry, err = discovery.NewRegistry(i.Node.NatsConn())
	if err != nil {
		log.Errorf("%v", err)
		return err
	}

	i.redis = db.NewRedis(conf.Redis)
	if i.redis == nil {
		return errors.New("new redis error")
	}

	i.s = newISLBServer(i, i.redis)
	pb.RegisterISLBServer(i.Node.ServiceRegistrar(), i.s)

	i.registry.Listen(i.s.handleNodeDiscovery)

	node := discovery.Node{
		DC:      conf.Global.Dc,
		Service: proto.ServiceISLB,
		NID:     i.Node.NID,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     conf.Nats.URL,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
	}

	go i.Node.KeepAlive(node)

	return nil
}

// Close all
func (i *ISLB) Close() {
	i.Node.Close()
	if i.redis != nil {
		i.redis.Close()
	}
	if i.registry != nil {
		i.registry.Close()
	}
}
