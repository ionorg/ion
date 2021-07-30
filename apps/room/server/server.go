package server

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	ndc "github.com/cloudwebrtc/nats-discovery/pkg/client"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	room "github.com/pion/ion/apps/biz/proto"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	islb "github.com/pion/ion/proto/islb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// BizServer represents an BizServer instance
type BizServer struct {
	room.UnimplementedRoomServer
	nc       *nats.Conn
	roomLock sync.RWMutex
	rooms    map[string]*Room
	closed   chan struct{}
	ndc      *ndc.Client
	islbcli  islb.ISLBClient
	bn       *BIZ
	stream   islb.ISLB_WatchISLBEventClient
}

// newBizServer creates a new avp server instance
func newBizServer(bn *BIZ, c string, nid string, nc *nats.Conn) (*BizServer, error) {

	ndc, err := ndc.NewClient(nc)
	if err != nil {
		log.Errorf("failed to create discovery client: %v", err)
		ndc.Close()
		return nil, err
	}

	b := &BizServer{
		ndc:    ndc,
		bn:     bn,
		nc:     nc,
		rooms:  make(map[string]*Room),
		closed: make(chan struct{}),
		stream: nil,
	}

	return b, nil
}

func (s *BizServer) close() {
	close(s.closed)
}

func (s *BizServer) Join(ctx context.Context, in *room.JoinRequest) (*room.JoinReply, error) {
	success := false
	reason := "unkown error."
	sid := in.Sid
	uid := in.Uid
	info := in.ExtraInfo
	r := s.getRoom(sid)

	if r == nil {
		reason = fmt.Sprintf("room sid = %v not found", sid)
		resp, err := s.ndc.Get(proto.ServiceRTC, map[string]interface{}{"sid": sid, "uid": uid})
		if err != nil {
			log.Errorf("dnc.Get: serivce = %v error %v", proto.ServiceRTC, err)
		}
		nid := ""
		if err == nil && len(resp.Nodes) > 0 {
			nid = resp.Nodes[0].NID
			r = s.createRoom(sid, nid)
			err = s.watchISLBEvent(nid, sid)
			if err != nil {
				log.Errorf("s.watchISLBEvent(req) failed %v", err)
			}
		} else {
			reason = "get serivce [sfu], node cnt == 0"
		}
	}

	if r != nil {
		peer := NewPeer(sid, uid, info /*repCh*/, nil) //TODO
		r.addPeer(peer)
		success = true
		reason = "join success."

		//Generate necessary metadata for routing.
		header := metadata.New(map[string]string{"service": "sfu", "nid": r.nid, "sid": sid, "uid": uid})
		err := stream.SendHeader(header)
		if err != nil {
			log.Errorf("stream.SendHeader failed %v", err)
		}
	}

	reply := &room.JoinReply{
		Config: &room.Configuration{
			Uuid:            "",
			IsAudioOnlyMode: false,
			Role:            room.Role_RoleHost,
		},
		Success: true,
	}
	return reply, nil
}

func (s *BizServer) Leave(ctx context.Context, in *room.LeaveRequest) (*room.LeaveReply, error) {
	uid := in.Uid
	sid := in.Sid
	r := s.getRoom(sid)
	if r == nil {
		return &room.LeaveReply{
			Success: false,
			Error: &room.Error{
				Code:   int32(codes.Internal),
				Reason: "room not exist",
			},
		}, status.Errorf(codes.Internal, "room not exist")
	}
	peer := r.getPeer(uid)
	if peer != nil && peer.uid == uid {
		if r.delPeer(peer) == 0 {
			s.delRoom(r)
			r = nil
		}
		peer.Close()
		peer = nil

	}

	return &room.LeaveReply{
		Success: true, Error: &room.Error{
			Code:   int32(codes.OK),
			Reason: "",
		},
	}, nil
}

func (s *BizServer) GetParticipants(ctx context.Context, in *room.Empty) (*room.GetParticipantsReply, error) {
	//TODO
	return nil, nil
}

func (s *BizServer) ReceiveNotification(in *room.Empty, server room.Room_ReceiveNotificationServer) error {

	//TODO
	return nil
}

func (s *BizServer) SetImportance(ctx context.Context, in *room.SetImportanceRequest) (*room.SetImportanceReply, error) {

	//TODO
	return nil, nil
}

func (s *BizServer) LockConference(ctx context.Context, in *room.LockConferenceRequest) (*room.LockConferenceReply, error) {

	//TODO
	return nil, nil
}

func (s *BizServer) EndConference(ctx context.Context, in *room.EndConferenceRequest) (*room.EndConferenceReply, error) {

	//TODO
	return nil, nil
}

func (s *BizServer) EditParticipantInfo(ctx context.Context, in *room.EditParticipantInfoRequest) (*room.EditParticipantInfoReply, error) {

	//TODO
	return nil, nil
}

func (s *BizServer) AddParticipant(ctx context.Context, in *room.AddParticipantRequest) (*room.AddParticipantReply, error) {

	//TODO
	return nil, nil
}

func (s *BizServer) RemoveParticipant(ctx context.Context, in *room.RemoveParticipantRequest) (*room.RemoveParticipantReply, error) {

	//TODO
	return nil, nil
}

func (s *BizServer) SendMessage(ctx context.Context, in *room.SendMessageRequest) (*room.SendMessageReply, error) {
	msg := in.Message
	sid := msg.Sid
	log.Debugf("Message: %+v", msg)
	// message broadcast
	r := s.getRoom(sid)
	if r == nil {
		log.Warnf("room not found, maybe the peer did not join")
		return &room.SendMessageReply{}, errors.New("room not exist")
	}
	r.sendMessage(in.Message)
	return &room.SendMessageReply{}, nil
}

func (s *BizServer) createRoom(sid string, sfuNID string) *Room {
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	if r := s.rooms[sid]; r != nil {
		return r
	}
	r := newRoom(sid, sfuNID)
	s.rooms[sid] = r
	return r
}

func (s *BizServer) getRoom(id string) *Room {
	s.roomLock.RLock()
	defer s.roomLock.RUnlock()
	return s.rooms[id]
}

func (s *BizServer) delRoom(r *Room) {
	id := r.SID()
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	if s.rooms[id] == r {
		delete(s.rooms, id)
	}
}

func (s *BizServer) watchISLBEvent(nid string, sid string) error {
	//s.islbLock.Lock()
	//defer s.islbLock.Unlock()

	if s.islbcli == nil {
		ncli, err := s.bn.NewNatsRPCClient(proto.ServiceISLB, nid, map[string]interface{}{})
		if err != nil {
			return err
		}
		s.islbcli = islb.NewISLBClient(ncli)
	}

	if s.stream == nil && s.islbcli != nil {
		stream, err := s.islbcli.WatchISLBEvent(context.Background())
		if err != nil {
			return err
		}
		err = stream.Send(&islb.WatchRequest{
			Nid: nid,
			Sid: sid,
		})
		if err != nil {
			return err
		}

		go func() {
			defer func() {
				//s.islbLock.Lock()
				//defer s.islbLock.Unlock()
				s.stream = nil
			}()

			for {
				req, err := stream.Recv()
				if err != nil {
					log.Errorf("BizServer.Singal server stream.Recv() err: %v", err)
					return
				}
				log.Infof("watchISLBEvent req => %v", req)
				switch payload := req.Payload.(type) {
				case *islb.ISLBEvent_Stream:
					r := s.getRoom(payload.Stream.Sid)
					if r != nil {
						r.sendStreamEvent(payload.Stream)
						p := r.getPeer(payload.Stream.Uid)
						// save last stream info.
						if p != nil {
							p.lastStreamEvent = payload.Stream
						}
					}
				}
			}
		}()

		s.stream = stream
	}
	return nil
}

// stat peers
func (s *BizServer) stat() {
	t := time.NewTicker(util.DefaultStatCycle)
	defer t.Stop()
	for {
		select {
		case <-t.C:
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
