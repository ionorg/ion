package islb

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/db"
	proto "github.com/pion/ion/pkg/grpc/islb"
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
	nc       *nats.Conn
	nodeLock sync.RWMutex
	s        *islbServer
	ngrpc    *rpc.Server
	registry *discovery.Registry
	redis    *db.Redis
}

// NewISLB create a islb node instance
func NewISLB() *ISLB {
	return &ISLB{}
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

	// connect options
	opts := []nats.Option{nats.Name("nats ion service")}
	opts = setupConnOptions(opts)

	// connect to nats server
	if i.nc, err = nats.Connect(conf.Nats.URL, opts...); err != nil {
		return err
	}

	//registry for node discovery.
	i.registry, err = discovery.NewRegistry(i.nc)
	if err != nil {
		log.Errorf("%v", err)
		return err
	}
	i.registry.Listen()

	i.redis = db.NewRedis(conf.Redis)
	if i.redis == nil {
		return errors.New("new redis error")
	}

	i.s = &islbServer{Redis: i.redis}
	//grpc service
	i.ngrpc = rpc.NewServer(i.nc, "islb")
	proto.RegisterISLBServer(i.ngrpc, i.s)

	return nil
}

// Close all
func (i *ISLB) Close() {
	if i.ngrpc != nil {
		i.ngrpc.Stop()
	}
	if i.nc != nil {
		i.nc.Close()
	}
	if i.redis != nil {
		i.redis.Close()
	}
}
