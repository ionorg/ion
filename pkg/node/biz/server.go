package biz

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	biz "github.com/pion/ion/pkg/grpc/biz"
	islb "github.com/pion/ion/pkg/grpc/islb"
	sfu "github.com/pion/ion/pkg/grpc/sfu"
	"github.com/pion/ion/pkg/proto"
)

const (
	statCycle = time.Second * 3
)

// BizServer represents an BizServer instance
type BizServer struct {
	biz.UnimplementedBizServer
	sfu.UnimplementedSFUServer
	nc       *nats.Conn
	elements []string
	roomLock sync.RWMutex
	rooms    map[string]*room
	closed   chan bool
	islbcli  islb.ISLBClient
	nodeLock sync.RWMutex
	nodes    map[string]*discovery.Node
}

// newBizServer creates a new avp server instance
func newBizServer(dc string, nid string, elements []string, nc *nats.Conn) *BizServer {
	return &BizServer{
		nc:       nc,
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
	var peer *Peer = nil
	for {
		req, err := stream.Recv()
		if err != nil {
			log.Errorf("err: %v", err)
			return err
		}

		switch payload := req.Payload.(type) {
		case *sfu.SignalRequest_Join:
			sid := payload.Join.Sid
			uid := payload.Join.Uid
			room := s.getRoom(sid)
			if room != nil {
				peer = room.getPeer(uid)
				if peer != nil {

				}
			}
		}

		log.Infof("req => %v", req.String())
	}
}

//Join for biz request.
func (s *BizServer) Join(stream biz.Biz_JoinServer) error {
	var r *room = nil
	var peer *Peer = nil
	defer func() {
		if peer != nil && r != nil {
			r.delPeer(peer.UID())
		}
	}()

	for {
		req, err := stream.Recv()
		if err != nil {
			log.Errorf("err: %v", err)
			return err
		}

		switch payload := req.Payload.(type) {
		case *biz.JoinRequest_Join:
			sid := payload.Join.Sid
			uid := payload.Join.Uid
			info := payload.Join.Info
			r = s.getRoom(sid)

			if r == nil {
				r = newRoom(sid)
				s.roomLock.RLock()
				s.rooms[sid] = r
				s.roomLock.RUnlock()
			}

			if s.islbcli == nil {
				ncli := nrpc.NewClient(s.nc, s.nodes["islb"].NID)
				s.islbcli = islb.NewISLBClient(ncli)
			}

			resp, err := s.islbcli.FindNode(context.TODO(), &islb.FindNodeRequest{
				Service: proto.ServiceSFU,
				Sid:     sid,
			})
			log.Infof("resp => %v", resp)

			if err != nil {
				stream.Send(&biz.JoinReply{
					Payload: &biz.JoinReply_Result{
						Result: &biz.JoinResult{
							Success: false,
							Reason:  "islb " + sid + " not found.",
						},
					},
				})
				break
			}

			if r != nil {
				peer = NewPeer(sid, uid, info)
				r.addPeer(peer)
				stream.Send(&biz.JoinReply{
					Payload: &biz.JoinReply_Result{
						Result: &biz.JoinResult{
							Success: true,
							Reason:  "join success.",
						},
					},
				})
			} else {
				stream.Send(&biz.JoinReply{
					Payload: &biz.JoinReply_Result{
						Result: &biz.JoinResult{
							Success: false,
							Reason:  "sid " + sid + " not found.",
						},
					},
				})
			}
		case *biz.JoinRequest_Leave:
			//uid := payload.Leave.Uid
			break

		case *biz.JoinRequest_Msg:
			//msg := payload.Msg
			//broadcast massge to room.
		}
		log.Infof("req => %v", req.String())
	}
}

// watchNodes watch islb nodes up/down
func (s *BizServer) watchNodes(state discovery.NodeState, node *discovery.Node) {
	s.nodeLock.Lock()
	defer s.nodeLock.Unlock()
	id := node.NID
	service := node.Service
	if state == discovery.NodeUp {
		log.Infof("Service up: "+service+" node id => [%v], rpc => %v", id, node.RPC.Protocol)
		if _, found := s.nodes[id]; !found {
			s.nodes[id] = node
		}
	} else if state == discovery.NodeDown {
		log.Infof("Service down: "+service+" node id => [%v]", id)
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
