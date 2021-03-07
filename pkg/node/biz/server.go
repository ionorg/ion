package biz

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	biz "github.com/pion/ion/pkg/grpc/biz"
	islb "github.com/pion/ion/pkg/grpc/islb"
	sfu "github.com/pion/ion/pkg/grpc/sfu"
)

const (
	statCycle = time.Second * 3
)

// BizServer represents an BizServer instance
type BizServer struct {
	biz.UnimplementedBizServer
	sfu.UnimplementedSFUServer
	elements []string
	roomLock sync.RWMutex
	rooms    map[string]*room
	closed   chan bool
	islbcli  islb.ISLBClient
	nodeLock sync.RWMutex
	nodes    map[string]*discovery.Node
}

// newBizServer creates a new avp server instance
func newBizServer(dc string, nid string, elements []string) *BizServer {
	return &BizServer{
		elements: elements,
		rooms:    make(map[string]*room),
		closed:   make(chan bool),
		nodes:    make(map[string]*discovery.Node),
	}
}

func (s *BizServer) start() error {
	//go s.stat()
	return nil
}

func (s *BizServer) close() {
	close(s.closed)
}

// getRoom get a room by id
func (s *BizServer) getRoom(id string) *room {
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	r := s.rooms[id]
	return r
}

//Signal forward to sfu node.
func (s *BizServer) Signal(stream sfu.SFU_SignalServer) error {
	for {
		payload, err := stream.Recv()
		if err != nil {
			log.Errorf("err: %v", err)
			return err
		}

		log.Infof("req => %v", payload.String())
	}
}

//Join for biz request.
func (s *BizServer) Join(stream biz.Biz_JoinServer) error {
	for {
		payload, err := stream.Recv()
		if err != nil {
			log.Errorf("err: %v", err)
			return err
		}

		log.Infof("req => %v", payload.String())
	}
}

// watchNodes watch islb nodes up/down
func (s *BizServer) watchIslbNodes(state discovery.NodeState, node *discovery.Node) {
	s.nodeLock.Lock()
	defer s.nodeLock.Unlock()
	id := node.NID
	if state == discovery.NodeUp {
		log.Infof("islb node %v up", id)
		if _, found := s.nodes[id]; !found {
			s.nodes[id] = node
		}
	} else if state == discovery.NodeDown {
		log.Infof("islb node %v down", id)
		delete(s.nodes, id)
	}
}

func (s *BizServer) getNode(service string, uid string, sid string) (string, error) {
	resp, err := s.islbcli.FindNode(context.Background(), &islb.FindNodeRequest{
		Service: service,
		Sid:     sid,
	})

	if err != nil {
		return "", err
	}
	return resp.Nodes[0].Nid, nil
}

// stat peers
func (s *BizServer) stat() {
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

/*
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

func (s *Server) handleSFUMessage(sid string, uid proto.UID, msg interface{}) {
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

	var sid string
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
func (s *Server) delPeer(sid string, uid proto.UID) {
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
func (s *Server) addPeer(sid string, peer *Peer) {
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
// func (s *server) getPeer(sid string, uid proto.UID) *peer {
// 	log.Infof("getPeer sid=%s uid=%s", sid, uid)
// 	r := s.getRoom(sid)
// 	if r == nil {
// 		log.Infof("room not exits, sid=%s uid=%s", sid, uid)
// 		return nil
// 	}
// 	return r.getPeer(uid)
// }

*/
