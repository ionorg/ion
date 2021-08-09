package server

import (
	"errors"
	"fmt"
	"sync"
	"time"

	log "github.com/pion/ion-log"
	room "github.com/pion/ion/apps/room/proto"
	"github.com/pion/ion/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BizServer represents an BizServer instance
type RoomService struct {
	room.UnimplementedRoomServer
	roomLock sync.RWMutex
	rooms    map[string]*Room
	closed   chan struct{}
}

// newBizServer creates a new avp server instance
func NewRoomService() (*RoomService, error) {
	b := &RoomService{
		rooms:  make(map[string]*Room),
		closed: make(chan struct{}),
	}

	return b, nil
}

func (s *RoomService) close() {
	close(s.closed)
}

func (s *RoomService) Signal(stream room.Room_SignalServer) error {
	participant := participant{
		sig: stream,
		ctx: stream.Context(),
	}

	defer participant.Close()

	req, err := stream.Recv()
	if err != nil {
		log.Errorf("RoomService.Singal server stream.Recv() err: %v", err)
		return err
	}

	switch payload := req.Payload.(type) {
	case *room.Request_Join:
		reply, err := s.Join(payload)
		if err != nil {
			log.Errorf("Join err: %v", err)
			return err
		}
		stream.Send(&room.Reply{Payload: reply})
	case *room.Request_Leave:
		reply, err := s.Leave(payload)
		if err != nil {
			log.Errorf("Leave err: %v", err)
			return err
		}
		stream.Send(&room.Reply{Payload: reply})
	case *room.Request_MediaPresentation:
		reply, err := s.MediaPresentation(payload)
		if err != nil {
			log.Errorf("LockConference err: %v", err)
			return err
		}
		stream.Send(&room.Reply{Payload: reply})
	}

	return nil
}

func (s *RoomService) Join(in *room.Request_Join) (*room.Reply_Join, error) {
	sid := in.Join.Sid
	uid := in.Join.Uid
	info := in.Join.ExtraInfo
	r := s.getRoom(sid)

	if r == nil {
		r = s.createRoom(sid, "todo nid")
	}

	if r != nil {
		peer := NewPeer(sid, uid, info)
		r.addPeer(peer)
		/*
			//Generate necessary metadata for routing.
			header := metadata.New(map[string]string{"service": "sfu", "nid": r.nid, "sid": sid, "uid": uid})
			err := stream.SendHeader(header)
			if err != nil {
				log.Errorf("stream.SendHeader failed %v", err)
			}
		*/
	}

	reply := &room.Reply_Join{
		Join: &room.JoinReply{
			Success: true,
			Error:   nil,
		},
	}
	return reply, nil
}

func (s *RoomService) Leave(in *room.Request_Leave) (*room.Reply_Leave, error) {
	uid := in.Leave.Sid
	sid := in.Leave.Uid
	r := s.getRoom(sid)
	if r == nil {
		return &room.Reply_Leave{
			Leave: &room.LeaveReply{
				Success: false,
				Error: &room.Error{
					Code:   room.ErrorType_RoomNotExist,
					Reason: "room not exist",
				},
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
	return &room.Reply_Leave{
		Leave: &room.LeaveReply{
			Success: true,
			Error:   nil,
		},
	}, nil
}

func (s *RoomService) GetParticipants(in *room.GetParticipantsRequest) (*room.GetParticipantsReply, error) {
	//TODO
	return nil, nil
}

func (s *RoomService) SetImportance(in *room.SetImportanceRequest) (*room.SetImportanceReply, error) {

	//TODO
	return nil, nil
}

func (s *RoomService) MediaPresentation(in *room.Request_MediaPresentation) (*room.Reply_MediaPresentation, error) {
	return nil, nil
}

func (s *RoomService) LockConference(in *room.LockConferenceRequest) (*room.LockConferenceReply, error) {

	//TODO
	return nil, nil
}

func (s *RoomService) EndConference(in *room.EndConferenceRequest) (*room.EndConferenceReply, error) {

	//TODO
	return nil, nil
}

func (s *RoomService) EditParticipantInfo(in *room.EditParticipantInfoRequest) (*room.EditParticipantInfoReply, error) {

	//TODO
	return nil, nil
}

func (s *RoomService) AddParticipant(in *room.AddParticipantRequest) (*room.AddParticipantReply, error) {

	//TODO
	return nil, nil
}

func (s *RoomService) RemoveParticipant(in *room.RemoveParticipantRequest) (*room.RemoveParticipantReply, error) {

	//TODO
	return nil, nil
}

func (s *RoomService) SendMessage(in *room.Request_SendMessage) (*room.Reply_SendMessage, error) {
	msg := in.SendMessage.Message
	sid := msg.Origin
	log.Debugf("Message: %+v", msg)
	// message broadcast
	r := s.getRoom(sid)
	if r == nil {
		log.Warnf("room not found, maybe the peer did not join")
		return &room.Reply_SendMessage{}, errors.New("room not exist")
	}
	r.sendMessage(msg)
	return &room.Reply_SendMessage{}, nil
}

func (s *RoomService) createRoom(sid string, sfuNID string) *Room {
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	if r := s.rooms[sid]; r != nil {
		return r
	}
	r := newRoom(sid, sfuNID)
	s.rooms[sid] = r
	return r
}

func (s *RoomService) getRoom(id string) *Room {
	s.roomLock.RLock()
	defer s.roomLock.RUnlock()
	return s.rooms[id]
}

func (s *RoomService) delRoom(r *Room) {
	id := r.SID()
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	if s.rooms[id] == r {
		delete(s.rooms, id)
	}
}

// stat peers
func (s *RoomService) stat() {
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
