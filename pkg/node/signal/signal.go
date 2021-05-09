package signal

import (
	"context"
	"fmt"
	"strings"

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
	}, nil
}

func (s *Signal) watchServiceDown(ctx context.Context, svc string, nid string, cli *nrpc.Client) {
	ndc, err := dc.NewClient(s.nc)
	if err != nil {
		log.Errorf("failed to create discovery client: %v", err)
		ndc.Close()
	}
	ndc.Watch(svc, func(state discovery.NodeState, node *discovery.Node) {
		if state == discovery.NodeDown && node.NID == nid {
			log.Infof("Service down: [%v]", svc)
			cli.Close()
		}
	})
	go func() {
		<-ctx.Done()
		log.Infof("Client down: [%v]", svc)
		ndc.Close()
	}()
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
			//TODO: using grpc.Metadata as Get parameters.
			resp, err := s.ndc.Get(svc, map[string]interface{}{})
			if err != nil || len(resp.Nodes) == 0 {
				log.Errorf("failed to Get service [%v]: %v", svc, err)
				return ctx, nil, status.Errorf(codes.Unavailable, "Service Unavailable")
			}
			nid := resp.Nodes[0].NID
			cli := nrpc.NewClient(s.nc, nid)
			s.watchServiceDown(ctx, svc, nid, cli)
			return ctx, cli, nil
		}
	}

	return ctx, nil, status.Errorf(codes.Unimplemented, "Unknown method")
}

func (s *Signal) Close() {
	s.nc.Close()
	s.ndc.Close()
}
