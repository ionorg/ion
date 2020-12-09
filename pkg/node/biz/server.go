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

// Server represents an Server instance
type Server struct {
	dc       string
	nid      string
	elements []string
	sub      *nats.Subscription
	nrpc     *proto.NatsRPC
	islb     string
	getNodes func() map[string]discovery.Node
	roomLock sync.RWMutex
	rooms    map[proto.SID]*room
	closed   chan bool
}

// newServer creates a new avp server instance
func newServer(dc string, nid string, elements []string, nrpc *proto.NatsRPC, getNodes func() map[string]discovery.Node) *Server {
	return &Server{
		dc:       dc,
		nid:      nid,
		nrpc:     nrpc,
		elements: elements,
		islb:     proto.ISLB(dc),
		getNodes: getNodes,
		rooms:    make(map[proto.SID]*room),
		closed:   make(chan bool),
	}
}

func (s *Server) start() error {
	var err error

	if s.sub, err = s.nrpc.Subscribe(s.nid, s.handle); err != nil {
		return err
	}

	go s.stat()

	return nil
}

func (s *Server) close() {
	close(s.closed)

	if s.sub != nil {
		if err := s.sub.Unsubscribe(); err != nil {
			log.Errorf("unsubscribe %s error: %v", s.sub.Subject, err)
		}
	}
}

func (s *Server) handle(msg interface{}) (interface{}, error) {
	log.Infof("handle incoming message: %T, %+v", msg, msg)

	switch v := msg.(type) {
	case *proto.SfuOfferMsg:
		s.handleSFUMessage(v.SID, v.UID, msg)
	case *proto.SfuTrickleMsg:
		s.handleSFUMessage(v.SID, v.UID, msg)
	case *proto.SfuICEConnectionStateMsg:
		s.handleSFUMessage(v.SID, v.UID, msg)
	default:
		log.Warnf("unkonw message: %v", msg)
	}

	return nil, nil
}

func (s *Server) handleSFUMessage(sid proto.SID, uid proto.UID, msg interface{}) {
	if r := s.getRoom(sid); r != nil {
		if p := r.getPeer(uid); p != nil {
			p.handleSFUMessage(msg)
		} else {
			log.Warnf("peer not exits, sid=%s, uid=%s", sid, uid)
		}
	} else {
		log.Warnf("room not exits, sid=%s, uid=%s", sid, uid)
	}
}

func (s *Server) broadcast(msg interface{}) (interface{}, error) {
	log.Infof("handle islb message: %T, %+v", msg, msg)

	var sid proto.SID
	var uid proto.UID

	switch v := msg.(type) {
	case *proto.FromIslbStreamAddMsg:
		sid, uid = v.SID, v.UID
	case *proto.ToClientPeerJoinMsg:
		sid, uid = v.SID, v.UID
	case *proto.IslbPeerLeaveMsg:
		sid, uid = v.SID, v.UID
	case *proto.IslbBroadcastMsg:
		sid, uid = v.SID, v.UID
	default:
		log.Warnf("unkonw message: %v", msg)
	}

	if r := s.getRoom(sid); r != nil {
		r.send(msg, uid)
	} else {
		log.Warnf("room not exits, sid=%s, uid=%s", sid, uid)
	}

	return nil, nil
}

func (s *Server) getNode(service string, uid proto.UID, sid proto.SID, mid proto.MID) (string, error) {
	resp, err := s.nrpc.Request(s.islb, proto.ToIslbFindNodeMsg{
		Service: service,
		UID:     uid,
		SID:     sid,
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
func (s *Server) getRoom(id proto.SID) *room {
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
func (s *Server) delPeer(sid proto.SID, uid proto.UID) {
	log.Infof("delPeer sid=%s uid=%s", sid, uid)
	room := s.getRoom(sid)
	if room == nil {
		log.Warnf("room not exits, sid=%s, uid=%s", sid, uid)
		return
	}
	if room.delPeer(uid) == 0 {
		s.roomLock.RLock()
		delete(s.rooms, sid)
		s.roomLock.RUnlock()
	}
}

// addPeer add a peer to room
func (s *Server) addPeer(sid proto.SID, peer *Peer) {
	log.Infof("addPeer sid=%s uid=%s", sid, peer.uid)
	room := s.getRoom(sid)
	if room == nil {
		room = newRoom(sid)
		s.roomLock.Lock()
		s.rooms[sid] = room
		s.roomLock.Unlock()
	}
	room.addPeer(peer)
}

// // getPeer get a peer in the room
// func (s *server) getPeer(sid proto.SID, uid proto.UID) *peer {
// 	log.Infof("getPeer sid=%s uid=%s", sid, uid)
// 	r := s.getRoom(sid)
// 	if r == nil {
// 		log.Infof("room not exits, sid=%s uid=%s", sid, uid)
// 		return nil
// 	}
// 	return r.getPeer(uid)
// }

// stat peers
func (s *Server) stat() {
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
		for sid, room := range s.rooms {
			info += fmt.Sprintf("room: %s\npeers: %d\n", sid, room.count())
		}
		s.roomLock.RUnlock()
		if len(info) > 0 {
			log.Infof("\n----------------signal-----------------\n" + info)
		}
	}
}
