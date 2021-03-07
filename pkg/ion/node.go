package ion

import (
	nd "github.com/cloudwebrtc/nats-discovery/pkg/client"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/util"
	"google.golang.org/grpc"
)

type Node struct {
	// Node ID
	NID string
	// Nats Client Conn
	nc *nats.Conn
	// gRPC Service Registrar
	nrpc *rpc.Server
	// Service discovery client
	nd *nd.Client
}

func (n *Node) Start(natURL string) error {
	var err error
	n.nc, err = util.NewNatsConn(natURL)
	if err != nil {
		log.Errorf("new nats conn error %v", err)
		n.Close()
		return err
	}
	n.nd, err = nd.NewClient(n.nc)
	if err != nil {
		log.Errorf("new discovery client error %v", err)
		n.Close()
		return err
	}
	n.nrpc = rpc.NewServer(n.nc, n.NID)
	return nil
}

func (n *Node) NatsConn() *nats.Conn {
	return n.nc
}

func (n *Node) KeepAlive(node discovery.Node) error {
	return n.nd.KeepAlive(node)
}

func (n *Node) Watch(service string, onStateChange nd.NodeStateChangeCallback) error {
	resp, err := n.nd.Get(service)
	if err != nil {
		log.Errorf("Watch service %v error %v", service, err)
		return err
	}
	for _, node := range resp.Nodes {
		onStateChange(discovery.NodeUp, &node)
	}

	return n.nd.Watch(service, onStateChange)
}

func (n *Node) ServiceRegistrar() grpc.ServiceRegistrar {
	return n.nrpc
}

func (n *Node) Close() {
	if n.nrpc != nil {
		n.nrpc.Stop()
	}
	if n.nc != nil {
		n.nc.Close()
	}
	if n.nd != nil {
		n.nd.Close()
	}
}
