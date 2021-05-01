package ion

import (
	"sync"

	nd "github.com/cloudwebrtc/nats-discovery/pkg/client"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/util"
	"google.golang.org/grpc"
)

type Service struct {
	Name    string
	Methods []*reflection.MethodDescriptor
}

type NeighborNode struct {
	Info    *discovery.Node
	Service []*Service
}

//Node .
type Node struct {
	// Node ID
	NID string
	// Nats Client Conn
	nc *nats.Conn
	// gRPC Service Registrar
	nrpc *rpc.Server
	// Service discovery client
	nd *nd.Client

	nodeLock sync.RWMutex
	//neighbor nodes
	neighborNodes map[string]*NeighborNode
}

//NewNode .
func NewNode(nid string) Node {
	return Node{
		NID:           nid,
		neighborNodes: make(map[string]*NeighborNode),
	}
}

//Start .
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

//NatsConn .
func (n *Node) NatsConn() *nats.Conn {
	return n.nc
}

//KeepAlive Upload your node info to registry.
func (n *Node) KeepAlive(node discovery.Node) error {
	return n.nd.KeepAlive(node)
}

//Watch the neighbor nodes
func (n *Node) Watch(service string) error {
	resp, err := n.nd.Get(service, map[string]interface{}{})
	if err != nil {
		log.Errorf("Watch service %v error %v", service, err)
		return err
	}
	for _, node := range resp.Nodes {
		n.handleNeighborNodes(discovery.NodeUp, &node)
	}

	return n.nd.Watch(service, n.handleNeighborNodes)
}

// GetNeighborNodes get neighbor nodes.
func (n *Node) GetNeighborNodes() map[string]*NeighborNode {
	n.nodeLock.Lock()
	defer n.nodeLock.Unlock()
	return n.neighborNodes
}

// handleNeighborNodes handle nodes up/down
func (n *Node) handleNeighborNodes(state discovery.NodeState, node *discovery.Node) {
	n.nodeLock.Lock()
	defer n.nodeLock.Unlock()
	id := node.NID
	service := node.Service
	if state == discovery.NodeUp {
		log.Infof("Service up: "+service+" node id => [%v], rpc => %v", id, node.RPC.Protocol)
		if _, found := n.neighborNodes[id]; !found {
			//Get service reflection info and verify service availability.
			info, err := util.GetServiceInfo(n.nc, id)
			if err != nil {
				log.Errorf("Can't get service info for %v", id)
				return
			}
			var services []*Service
			for svc, mds := range info {
				services = append(services, &Service{
					Name:    svc,
					Methods: mds,
				})
			}
			log.Debugf("Service info for %v, info %v", id, info)

			neighborNode := &NeighborNode{
				Info:    node,
				Service: services,
			}
			n.neighborNodes[id] = neighborNode
		}
	} else if state == discovery.NodeDown {
		log.Infof("Service down: "+service+" node id => [%v]", id)
		delete(n.neighborNodes, id)
	}
}

//ServiceRegistrar return grpc.ServiceRegistrar of this node, used to create grpc services
func (n *Node) ServiceRegistrar() grpc.ServiceRegistrar {
	return n.nrpc
}

//Close .
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
