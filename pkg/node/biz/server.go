package biz

import (
	"errors"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
)

// server represents an server instance
type server struct {
	nid      string
	elements []string
	sub      *nats.Subscription
	sig      *signal
	nrpc     *proto.NatsRPC
	getNodes func() map[string]discovery.Node
}

// newServer creates a new avp server instance
func newServer(nid string, elements []string, nrpc *proto.NatsRPC, getNodes func() map[string]discovery.Node) *server {
	return &server{
		nid:      nid,
		nrpc:     nrpc,
		elements: elements,
		getNodes: getNodes,
	}
}

func (s *server) start(conf signalConf) error {
	var err error

	s.sig = newSignal(s)
	if err = s.sig.start(conf); err != nil {
		return err
	}

	if s.sub, err = s.nrpc.Subscribe(s.nid, s.handle); err != nil {
		return err
	}

	return nil
}

func (s *server) close() {
	if s.sub != nil {
		if err := s.sub.Unsubscribe(); err != nil {
			log.Errorf("unsubscribe %s error: %v", s.sub.Subject, err)
		}
	}
	if s.sig != nil {
		s.sig.close()
	}
}

func (s *server) handle(msg interface{}) (interface{}, error) {
	log.Infof("handle incoming message: %T, %+v", msg, msg)

	switch v := msg.(type) {
	case *proto.SfuOfferMsg:
		s.handleSFUMessage(v.RID, v.UID, msg)
	case *proto.SfuTrickleMsg:
		s.handleSFUMessage(v.RID, v.UID, msg)
	case *proto.SfuICEConnectionStateMsg:
		s.handleSFUMessage(v.RID, v.UID, msg)
	default:
		log.Warnf("unkonw message: %v", msg)
	}

	return nil, nil
}

func (s *server) handleSFUMessage(rid proto.RID, uid proto.UID, msg interface{}) {
	if r := getRoom(rid); r != nil {
		if p := r.getPeer(uid); p != nil {
			p.handleSFUMessage(msg)
		} else {
			log.Warnf("peer not exits, rid=%s, uid=%s", rid, uid)
		}
	} else {
		log.Warnf("room not exits, rid=%s, uid=%s", rid, uid)
	}
}

func (s *server) broadcast(msg interface{}) (interface{}, error) {
	log.Infof("handle islb message: %T, %+v", msg, msg)

	switch v := msg.(type) {
	case *proto.FromIslbStreamAddMsg:
		s.notifyRoom(proto.ClientOnStreamAdd, v.RID, v.UID, msg)
	case *proto.ToClientPeerJoinMsg:
		s.notifyRoom(proto.ClientOnJoin, v.RID, v.UID, msg)
	case *proto.IslbPeerLeaveMsg:
		s.notifyRoom(proto.ClientOnLeave, v.RID, v.UID, msg)
	case *proto.IslbBroadcastMsg:
		s.notifyRoom(proto.ClientBroadcast, v.RID, v.UID, msg)
	default:
		log.Warnf("unkonw message: %v", msg)
	}

	return nil, nil
}

func (s *server) notifyRoom(method string, rid proto.RID, withoutID proto.UID, msg interface{}) {
	log.Infof("broadcast: method=%s, msg=%v", method, msg)
	if r := getRoom(rid); r != nil {
		r.notifyWithoutID(method, msg, withoutID)
	} else {
		log.Warnf("room not exits, rid=%s, uid=%s", rid, withoutID)
	}
}

func (s *server) getIslb() string {
	nodes := s.getNodes()
	for _, item := range nodes {
		if item.Service == proto.ServiceISLB {
			return item.NID
		}
	}
	log.Warnf("not found islb")
	return ""
}

func (s *server) getNode(service string, islb string, uid proto.UID, rid proto.RID, mid proto.MID) (string, error) {
	if islb == "" {
		if islb = s.getIslb(); islb == "" {
			return "", errors.New("not found islb")
		}
	}

	resp, err := s.nrpc.Request(islb, proto.ToIslbFindNodeMsg{
		Service: service,
		UID:     uid,
		RID:     rid,
		MID:     mid,
	})

	if err != nil {
		return "", err
	}

	msg, ok := resp.(*proto.FromIslbFindNodeMsg)
	if !ok {
		return "", errors.New("parse islb-find-node msg error")
	}

	return msg.ID, nil
}
