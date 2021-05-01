package signal

import (
	"context"
	"fmt"
	"strings"
	"sync"

	dc "github.com/cloudwebrtc/nats-discovery/pkg/client"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
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

// Config for biz node
type Config struct {
	Global global     `mapstructure:"global"`
	Log    logConf    `mapstructure:"log"`
	Nats   natsConf   `mapstructure:"nats"`
	Avp    avpConf    `mapstructure:"avp"`
	Signal signalConf `mapstructure:"signal"`
}

type Signal struct {
	conf   Config
	nc     *nats.Conn
	ndc    *dc.Client
	rwlock sync.RWMutex
	svc    map[string]string
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
		svc:  make(map[string]string),
	}, nil
}

func (s *Signal) saveServiceInfo(svc string, state discovery.NodeState, node *discovery.Node) {
	switch state {
	case discovery.NodeUp:
		nid := node.NID
		log.Infof("svc %v => %v", svc, node)
		info, err := util.GetServiceInfo(s.nc, nid)
		if err != nil {
			log.Errorf("Can't get service info for %v", nid)
			return
		}
		for fullSvcName, mds := range info {
			log.Infof("fullSvcName: %v, mds %v", fullSvcName, mds)
			s.rwlock.Lock()
			defer s.rwlock.Unlock()
			s.svc[fullSvcName] = nid
		}
	}
}

func (s *Signal) Start() {
	for _, svc := range s.conf.Signal.SVC.Services {
		log.Infof("Watch svc %v", svc)
		resp, err := s.ndc.Get(svc, map[string]interface{}{})
		if err != nil {
			log.Errorf("Watch service %v error %v", svc, err)
			break
		}
		for _, node := range resp.Nodes {
			s.saveServiceInfo(svc, discovery.NodeUp, &node)
		}
		s.ndc.Watch(svc, func(state discovery.NodeState, node *discovery.Node) {
			log.Infof("svc %v => %v", svc, state)
			s.saveServiceInfo(svc, state, node)
		})
	}
}

func (s *Signal) Director(ctx context.Context, fullMethodName string) (context.Context, grpc.ClientConnInterface, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		log.Infof("md %v", md)
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

	//Find node id by existing node.
	s.rwlock.RLock()
	for svc, nid := range s.svc {
		if strings.HasPrefix(fullMethodName, "/"+svc) {
			cli := nrpc.NewClient(s.nc, nid)
			return ctx, cli, nil
		}
	}
	s.rwlock.RUnlock()

	//Find service in neighbor nodes.
	svcConf := s.conf.Signal.SVC
	for _, svc := range svcConf.Services {
		if strings.HasPrefix(fullMethodName, "/"+svc+".") {
			resp, err := s.ndc.Get(svc, map[string]interface{}{})
			if err != nil || len(resp.Nodes) == 0 {
				log.Errorf("failed to Get service [%v]: %v", svc, err)
				return ctx, nil, status.Errorf(codes.Unavailable, "Service Unavailable")
			}
			nid := resp.Nodes[0].NID
			cli := nrpc.NewClient(s.nc, nid)
			return ctx, cli, nil
		}
	}

	return ctx, nil, status.Errorf(codes.Unimplemented, "Unknown method")
}

func (s *Signal) Close() {
	s.nc.Close()
	s.ndc.Close()
}
