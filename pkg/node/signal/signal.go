package signal

import (
	"context"
	"fmt"
	"strings"

	dc "github.com/cloudwebrtc/nats-discovery/pkg/client"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type svcConf struct {
	Services []string `mapstructure:"services"`
}

type signalConf struct {
	GRPC grpcConf   `mapstructure:"grpc"`
	JWT  AuthConfig `mapstructure:"jwt"`
	SVC  svcConf    `mapstructure:"svc"`
}

// signalConf represents signal server configuration
type grpcConf struct {
	Host            string `mapstructure:"host"`
	Port            int    `mapstructure:"port"`
	Cert            string `mapstructure:"cert"`
	Key             string `mapstructure:"key"`
	AllowAllOrigins bool   `mapstructure:"allow_all_origins"`
}

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

type nodeConf struct {
	NID string `mapstructure:"nid"`
}

// Config for biz node
type Config struct {
	Global global     `mapstructure:"global"`
	Log    logConf    `mapstructure:"log"`
	Nats   natsConf   `mapstructure:"nats"`
	Node   nodeConf   `mapstructure:"node"`
	Avp    avpConf    `mapstructure:"avp"`
	Signal signalConf `mapstructure:"signal"`
}

type Signal struct {
	ion.Node
	conf Config
	nc   *nats.Conn
	ndc  *dc.Client
}

func NewSignal(conf Config) (*Signal, error) {
	nc, err := util.NewNatsConn(conf.Nats.URL)
	if err != nil {
		log.Errorf("new nats conn error %v", err)
		nc.Close()
		return nil, err
	}
	ndc, err := dc.NewClient(nc)
	if err != nil {
		log.Errorf("failed to create discovery client: %v", err)
		ndc.Close()
		return nil, err
	}
	return &Signal{
		conf: conf,
		nc:   nc,
		ndc:  ndc,
		Node: ion.NewNode(conf.Node.NID),
	}, nil
}

func (s *Signal) Start() error {
	err := s.Node.Start(s.conf.Nats.URL)
	if err != nil {
		s.Close()
		return err
	}
	node := discovery.Node{
		DC:      s.conf.Global.Dc,
		Service: proto.ServiceSIG,
		NID:     s.Node.NID,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     s.conf.Nats.URL,
			//Params:   map[string]string{"username": "foo", "password": "bar"},
		},
	}

	go func() {
		err := s.Node.KeepAlive(node)
		if err != nil {
			log.Errorf("sig.Node.KeepAlive(%v) error %v", s.Node.NID, err)
		}
	}()

	//Watch ALL nodes.
	go func() {
		err := s.Node.Watch(proto.ServiceALL)
		if err != nil {
			log.Errorf("Node.Watch(proto.ServiceALL) error %v", err)
		}
	}()

	return nil
}

func (s *Signal) Director(ctx context.Context, fullMethodName string) (context.Context, grpc.ClientConnInterface, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		log.Infof("fullMethodName: %v, md %v", fullMethodName, md)
	}

	//Authenticate here.
	authConfig := &s.conf.Signal.JWT
	if authConfig.Enabled {
		claims, err := getClaim(ctx, authConfig)
		if err != nil {
			return ctx, nil, status.Errorf(codes.Unauthenticated, fmt.Sprintf("Failed to Get Claims JWT : %v", err))
		}

		log.Infof("claims: UID: %s, SID: %v, Services: %v", claims.UID, claims.SID, claims.Services)

		allowed := false
		for _, svc := range claims.Services {
			if strings.Contains(fullMethodName, "/"+svc+".") {
				allowed = true
				break
			}
		}

		if !allowed {
			return ctx, nil, status.Errorf(codes.Unauthenticated, fmt.Sprintf("Service %v access denied!", fullMethodName))
		}
	}

	//Find service in neighbor nodes.
	svcConf := s.conf.Signal.SVC
	for _, svc := range svcConf.Services {
		if strings.HasPrefix(fullMethodName, "/"+svc+".") {
			//Using grpc.Metadata as a parameters for ndc.Get.
			var parameters = make(map[string]interface{})
			for key, value := range md {
				parameters[key] = value[0]
			}
			cli, err := s.NewNatsRPCClient(svc, "*", parameters)
			if err != nil {
				log.Errorf("failed to Get service [%v]: %v", svc, err)
				return ctx, nil, status.Errorf(codes.Unavailable, "Service Unavailable: %v", err)
			}
			return ctx, cli, nil
		}
	}

	return ctx, nil, status.Errorf(codes.Unimplemented, "Unknown Service.Method %v", fullMethodName)
}

func (s *Signal) Close() {
	s.nc.Close()
	s.ndc.Close()
}
