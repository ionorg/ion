package avp

import (
	"context"
	"io"
	"sync"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	avp "github.com/pion/ion-avp/pkg"
	log "github.com/pion/ion-log"
	pb "github.com/pion/ion/pkg/grpc/avp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AVPProcesser represents an avp instance
type AVPProcesser struct {
	config  avp.Config
	clients map[string]*SFU
	mu      sync.RWMutex
}

// NewAVPProcesser creates a new avp instance
func NewAVPProcesser(c avp.Config, elems map[string]avp.ElementFun) *AVPProcesser {
	a := &AVPProcesser{
		config:  c,
		clients: make(map[string]*SFU),
	}

	avp.Init(elems)

	return a
}

// Process starts a process for a track.
func (a *AVPProcesser) Process(ctx context.Context, addr, pid, sid, tid, eid string, config []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	c := a.clients[addr]
	// no client yet, create one
	if c == nil {
		var err error
		if c, err = NewSFU(addr, a.config); err != nil {
			return err
		}
		c.OnClose(func() {
			a.mu.Lock()
			defer a.mu.Unlock()
			delete(a.clients, addr)
		})
		a.clients[addr] = c
	}

	t, err := c.GetTransport(sid)
	if err != nil {
		return err
	}

	return t.Process(pid, tid, eid, config)
}

type avpServer struct {
	pb.UnimplementedAVPServer
	avp      *AVPProcesser
	nodeLock sync.RWMutex
	nodes    map[string]*discovery.Node
}

func newAVPServer(conf avp.Config, elems map[string]avp.ElementFun) *avpServer {
	return &avpServer{
		avp:   NewAVPProcesser(conf, elems),
		nodes: make(map[string]*discovery.Node),
	}
}

// watchIslbNodes watch islb nodes up/down
func (a *avpServer) watchIslbNodes(state discovery.NodeState, node *discovery.Node) {
	a.nodeLock.Lock()
	defer a.nodeLock.Unlock()
	id := node.NID
	if state == discovery.NodeUp {
		log.Infof("islb node %v up", id)
		if _, found := a.nodes[id]; !found {
			a.nodes[id] = node
		}
	} else if state == discovery.NodeDown {
		log.Infof("islb node %v down", id)
		delete(a.nodes, id)
	}
}

// Signal handler for avp server
func (s *avpServer) Signal(stream pb.AVP_SignalServer) error {
	for {
		in, err := stream.Recv()

		if err != nil {
			if err == io.EOF {
				return nil
			}

			errStatus, _ := status.FromError(err)
			if errStatus.Code() == codes.Canceled {
				return nil
			}

			log.Errorf("signal error %v %v", errStatus.Message(), errStatus.Code())
			return err
		}

		if payload, ok := in.Payload.(*pb.SignalRequest_Process); ok {
			if err = s.avp.Process(
				stream.Context(),
				payload.Process.Sfu,
				payload.Process.Pid,
				payload.Process.Sid,
				payload.Process.Tid,
				payload.Process.Eid,
				payload.Process.Config,
			); err != nil {
				log.Errorf("process error: %v", err)
			}
		}
	}
}
