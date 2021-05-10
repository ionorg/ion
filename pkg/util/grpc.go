package util

import (
	"fmt"
	"net"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	log "github.com/pion/ion-log"
	"google.golang.org/grpc"
)

// ClientConnInterface .
type ClientConnInterface interface {
	grpc.ClientConnInterface
	Close() error
}

// NewGRPCClientForNode .
func NewGRPCClientConnForNode(node discovery.Node) (ClientConnInterface, error) {
	switch node.RPC.Protocol {
	case discovery.GRPC:
		conn, err := grpc.Dial(node.RPC.Addr, grpc.WithInsecure(), grpc.WithBlock())
		if err != nil {
			log.Errorf("did not connect: %v", err)
			return nil, err
		}
		return conn, err
	case discovery.NGRPC:
		nc, err := NewNatsConn(node.RPC.Addr)
		if err != nil {
			log.Errorf("new nats conn error %v", err)
			return nil, err
		}
		conn := nrpc.NewClient(nc, node.NID, "unkown")
		return conn, nil
	case discovery.JSONRPC:
		return nil, fmt.Errorf("%v not yet implementation", node.RPC.Protocol)
	}
	return nil, fmt.Errorf("New grpc client failed")
}

// ServiceInterface .
type ServiceInterface interface {
	grpc.ServiceRegistrar
	Serve(lis net.Listener) error
	Stop()
}

type nrpcServer struct {
	*nrpc.Server
}

func (n *nrpcServer) Serve(lis net.Listener) error {
	return nil
}

//NewGRPCServiceForNode .
func NewGRPCServiceForNode(node discovery.Node) (ServiceInterface, error) {
	switch node.RPC.Protocol {
	case discovery.GRPC:
		lis, err := net.Listen("tcp", node.RPC.Addr)
		if err != nil {
			log.Panicf("failed to listen: %v", err)
		}
		log.Infof("--- GRPC Server Listening at %s ---", node.RPC.Addr)

		s := grpc.NewServer()
		if err := s.Serve(lis); err != nil {
			log.Panicf("failed to serve: %v", err)
			return nil, err
		}
		return s, err
	case discovery.NGRPC:
		nc, err := NewNatsConn(node.RPC.Addr)
		if err != nil {
			log.Errorf("new nats conn error %v", err)
			return nil, err
		}
		s := nrpc.NewServer(nc, node.NID)
		return &nrpcServer{s}, nil
	case discovery.JSONRPC:
		return nil, fmt.Errorf("%v not yet implementation", node.RPC.Protocol)
	}

	return nil, fmt.Errorf("New grpc server failed")
}
