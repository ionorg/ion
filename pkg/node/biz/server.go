package biz

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
)

const (
	statCycle = time.Second * 3
)

// server represents an server instance
type server struct {
	dc       string
	nid      string
	elements []string
	sub      *nats.Subscription
	sig      *signal
	nrpc     *proto.NatsRPC
	islb     string
	getNodes func() map[string]discovery.Node
	roomLock sync.RWMutex
	rooms    map[proto.RID]*room
	closed   chan bool
}

// newServer creates a new avp server instance
func newServer(dc string, nid string, elements []string, nrpc *proto.NatsRPC, getNodes func() map[string]discovery.Node) *server {
	return &server{
		dc:       dc,
		nid:      nid,
		nrpc:     nrpc,
		elements: elements,
		islb:     proto.ISLB(dc),
		getNodes: getNodes,
		rooms:    make(map[proto.RID]*room),
		closed:   make(chan bool),
	}
}

func (s *server) start(conf signalConf) error {
	var err error

	s.sig = newSignal(s)
	s.sig.start(conf)

	if s.sub, err = s.nrpc.Subscribe(s.nid, s.handle); err != nil {
		return err
	}

	go s.stat()

	return nil
}

func (s *server) close() {
	close(s.closed)

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
	if r := s.getRoom(rid); r != nil {
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
		s.notifyRoomWithoutID(proto.ClientOnStreamAdd, v.RID, v.UID, msg)
	case *proto.ToClientPeerJoinMsg:
		s.notifyRoomWithoutID(proto.ClientOnJoin, v.RID, v.UID, msg)
	case *proto.IslbPeerLeaveMsg:
		s.notifyRoomWithoutID(proto.ClientOnLeave, v.RID, v.UID, msg)
	case *proto.IslbBroadcastMsg:
		s.notifyRoomWithoutID(proto.ClientBroadcast, v.RID, v.UID, msg)
	default:
		log.Warnf("unkonw message: %v", msg)
	}

	return nil, nil
}

func (s *server) getNode(service string, uid proto.UID, rid proto.RID, mid proto.MID) (string, error) {
	resp, err := s.nrpc.Request(s.islb, proto.ToIslbFindNodeMsg{
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

// getRoom get a room by id
func (s *server) getRoom(id proto.RID) *room {
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	r := s.rooms[id]
	return r
}

// // getRoomsByPeer a peer in many room
// func (s *server) getRoomsByPeer(uid proto.UID) []*room {
// 	var result []*room
// 	s.roomLock.RLock()
// 	defer s.roomLock.RUnlock()
// 	for _, r := range s.rooms {
// 		if p := r.getPeer(uid); p != nil {
// 			result = append(result, r)
// 		}
// 	}
// 	return result
// }

// delPeer delete a peer in the room
func (s *server) delPeer(rid proto.RID, uid proto.UID) {
	log.Infof("delPeer rid=%s uid=%s", rid, uid)
	room := s.getRoom(rid)
	if room == nil {
		log.Warnf("room not exits, rid=%s, uid=%s", rid, uid)
		return
	}
	if room.delPeer(uid) == 0 {
		s.roomLock.RLock()
		delete(s.rooms, rid)
		s.roomLock.RUnlock()
	}
}

// addPeer add a peer to room
func (s *server) addPeer(rid proto.RID, peer *peer) {
	log.Infof("addPeer rid=%s uid=%s", rid, peer.uid)
	room := s.getRoom(rid)
	if room == nil {
		room = newRoom(rid)
		s.roomLock.Lock()
		s.rooms[rid] = room
		s.roomLock.Unlock()
	}
	room.addPeer(peer)
}

// // getPeer get a peer in the room
// func (s *server) getPeer(rid proto.RID, uid proto.UID) *peer {
// 	log.Infof("getPeer rid=%s uid=%s", rid, uid)
// 	r := s.getRoom(rid)
// 	if r == nil {
// 		log.Infof("room not exits, rid=%s uid=%s", rid, uid)
// 		return nil
// 	}
// 	return r.getPeer(uid)
// }

// // notifyPeer send message to peer
// func (s *server) notifyPeer(method string, rid proto.RID, uid proto.UID, data interface{}) {
// 	log.Infof("notifyPeer rid=%s, uid=%s, data=%s", rid, uid, data)
// 	room := s.getRoom(rid)
// 	if room == nil {
// 		log.Warnf("room not exits, rid=%s, uid=%s, data=%s", rid, uid, data)
// 		return
// 	}
// 	peer := room.getPeer(uid)
// 	if peer == nil {
// 		log.Warnf("peer not exits, rid=%s, uid=%s, data=%v", rid, uid, data)
// 		return
// 	}
// 	if err := peer.notify(method, data); err != nil {
// 		log.Errorf("notify peer error: %s, rid=%s, uid=%s, data=%v", err, rid, uid, data)
// 	}
// }

// notifyRoomWithoutID notify room
func (s *server) notifyRoomWithoutID(method string, rid proto.RID, withoutID proto.UID, msg interface{}) {
	log.Infof("broadcast: method=%s, msg=%v", method, msg)
	if r := s.getRoom(rid); r != nil {
		r.notifyWithoutID(method, msg, withoutID)
	} else {
		log.Warnf("room not exits, rid=%s, uid=%s", rid, withoutID)
	}
}

// stat peers
func (s *server) stat() {
	t := time.NewTicker(statCycle)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			break
		case <-s.closed:
			log.Infof("stop stat")
			return
		}

		var info string
		s.roomLock.RLock()
		for rid, room := range s.rooms {
			info += fmt.Sprintf("room: %s\npeers: %d\n", rid, room.count())
		}
		s.roomLock.RUnlock()
		if len(info) > 0 {
			log.Infof("\n----------------signal-----------------\n" + info)
		}
	}
}
