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
	// TODO: handle incoming message

	return nil, nil
}

func (s *server) broadcast(msg interface{}) (interface{}, error) {
	log.Infof("handle islb message: %T, %+v", msg, msg)

	var method string
	var rid proto.RID
	var uid proto.UID

	switch v := msg.(type) {
	case *proto.FromIslbStreamAddMsg:
		method, rid, uid = proto.ClientOnStreamAdd, v.RID, v.UID
	case *proto.ToClientPeerJoinMsg:
		method, rid, uid = proto.ClientOnJoin, v.RID, v.UID
	case *proto.IslbPeerLeaveMsg:
		method, rid, uid = proto.ClientOnLeave, v.RID, v.UID
	case *proto.IslbBroadcastMsg:
		method, rid, uid = proto.ClientBroadcast, v.RID, v.UID
	default:
		log.Warnf("unkonw message: %v", msg)
	}

	log.Infof("broadcast: method=%s, msg=%v", method, msg)
	if r := getRoom(rid); r != nil {
		go func(method string, msg interface{}, uid proto.UID) {
			r.notifyWithoutID(method, msg, uid)
		}(method, msg, uid)
	} else {
		log.Warnf("room not exits, rid=%s, uid=%s", rid, uid)
	}

	return nil, nil
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
