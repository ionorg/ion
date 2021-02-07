package avp

import (
	"context"
	"sync"

	"github.com/nats-io/nats.go"
	iavp "github.com/pion/ion-avp/pkg"
	log "github.com/pion/ion-log"
	proto "github.com/pion/ion/pkg/grpc/avp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type avpServer struct {
	proto.UnimplementedAVPServer
	config  iavp.Config
	clients map[string]*sfuClient
	nid     string
	nc      *nats.Conn
	mu      sync.RWMutex
}

func (s *avpServer) Process(context.Context, *proto.AVPRequest) (*proto.AVPReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Process not implemented")
}

// newAvpServer creates a new avp server instance
func newAvpServer(conf iavp.Config, elems map[string]iavp.ElementFun, nid string, nc *nats.Conn) *avpServer {
	s := &avpServer{
		config:  conf,
		clients: make(map[string]*sfuClient),
		nid:     nid,
		nc:      nc,
	}
	iavp.Init(elems)
	return s
}

func (s *avpServer) handle(msg interface{}) (interface{}, error) {
	log.Infof("handle incoming message: %T, %+v", msg, msg)

	switch v := msg.(type) {
	case *proto.ToAvpProcessMsg:
		if err := s.process(v.Addr, v.PID, v.SID, v.TID, v.EID, v.Config); err != nil {
			return nil, err
		}
	case *proto.SfuOfferMsg:
		s.handleSFUMessage(string(v.UID), msg)
	case *proto.SfuTrickleMsg:
		s.handleSFUMessage(string(v.UID), msg)
	case *proto.SfuICEConnectionStateMsg:
		s.handleSFUMessage(string(v.UID), msg)
	default:
		log.Warnf("unkonw message: %v", msg)
	}

	return nil, nil
}

func (s *avpServer) handleSFUMessage(addr string, msg interface{}) {
	s.mu.Lock()
	client := s.clients[addr]
	s.mu.Unlock()

	if client != nil {
		client.handleSFUMessage(msg)
	} else {
		log.Warnf("not found sfu client, addr=%s", addr)
	}
}

// process starts a process for a track.
func (s *avpServer) process(addr, pid, sid, tid, eid string, config []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	c := s.clients[addr]
	// no client yet, create one
	if c == nil {
		var err error
		log.Infof("create a sfu client, addr=%s", addr)
		if c, err = newSFU(addr, s.config, s.nid, s.nrpc); err != nil {
			return err
		}
		c.onClose(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			log.Infof("sfu client close, addr=%s", addr)
			delete(s.clients, addr)
		})
		s.clients[addr] = c
	} else {
		log.Infof("sfu client exist, addr=%s", addr)
	}

	t, err := c.getTransport(proto.SID(sid))
	if err != nil {
		return err
	}

	return t.Process(pid, tid, eid, config)
}
