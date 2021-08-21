package server

import (
	"errors"

	log "github.com/pion/ion-log"
	room "github.com/pion/ion/apps/room/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// BizServer represents an BizServer instance
type RoomSignalService struct {
	room.UnimplementedRoomSignalServer
	rs *RoomService
}

// NewRoomService creates a new room app server instance
func NewRoomSignalService(rs *RoomService) *RoomSignalService {
	s := &RoomSignalService{
		rs: rs,
	}
	return s
}

func (s *RoomSignalService) Signal(stream room.RoomSignal_SignalServer) error {
	var p *Peer = nil
	defer func() {
		if p != nil {
			p.Close()
		}
	}()

	for {
		req, err := stream.Recv()
		if err != nil {
			log.Errorf("RoomSignalService.Singal server stream.Recv() err: %v", err)
			return err
		}

		switch payload := req.Payload.(type) {
		case *room.Request_Join:
			reply, peer, err := s.Join(payload)
			if err != nil {
				log.Errorf("Join err: %v", err)
				return err
			}
			peer.sig = stream
			p = peer
			stream.Send(&room.Reply{Payload: reply})
		case *room.Request_Leave:
			reply, err := s.Leave(payload)
			if err != nil {
				log.Errorf("Leave err: %v", err)
				return err
			}
			stream.Send(&room.Reply{Payload: reply})
		case *room.Request_SendMessage:
			reply, err := s.SendMessage(payload)
			if err != nil {
				log.Errorf("LockConference err: %v", err)
				return err
			}
			stream.Send(&room.Reply{Payload: reply})
		}
	}
}

func (s *RoomSignalService) Join(in *room.Request_Join) (*room.Reply_Join, *Peer, error) {
	pinfo := in.Join.Peer
	sid := pinfo.Sid
	uid := pinfo.Uid

	var peer *Peer = nil
	r := s.rs.getRoom(sid)

	log.Infof("s.rs.getRoom======%+v sid=%v", r, sid)
	if r == nil {
		r = s.rs.createRoom(sid)
		/*
			reply := &room.Reply_Join{
				Join: &room.JoinReply{
					Success: false,
					Error: &room.Error{
						Code:   room.ErrorType_RoomNotExist,
						Reason: "room not exist",
					},
				},
			}
			return reply, nil, fmt.Errorf("room [%v] not exist", sid)
		*/
	}

	peer = NewPeer(uid)
	peer.info = *pinfo
	r.addPeer(peer)
	// TODO
	/*
		//Generate necessary metadata for routing.
		header := metadata.New(map[string]string{"service": "sfu", "nid": r.nid, "sid": sid, "uid": uid})
		err := stream.SendHeader(header)
		if err != nil {
			log.Errorf("stream.SendHeader failed %v", err)
		}
	*/

	// TODO get from db

	reply := &room.Reply_Join{
		Join: &room.JoinReply{
			Success: true,
			Room:    &r.info,
			Error:   nil,
		},
	}
	return reply, peer, nil
}

func (s *RoomSignalService) Leave(in *room.Request_Leave) (*room.Reply_Leave, error) {
	uid := in.Leave.Sid
	sid := in.Leave.Uid
	r := s.rs.getRoom(sid)
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
			s.rs.delRoom(r)
			r = nil
		}
	}
	return &room.Reply_Leave{
		Leave: &room.LeaveReply{
			Success: true,
			Error:   nil,
		},
	}, nil
}

func (s *RoomSignalService) SendMessage(in *room.Request_SendMessage) (*room.Reply_SendMessage, error) {
	msg := in.SendMessage.Message
	sid := in.SendMessage.Sid
	log.Debugf("Message: %+v", msg)
	r := s.rs.getRoom(sid)
	if r == nil {
		log.Warnf("room not found, maybe the peer did not join")
		return &room.Reply_SendMessage{}, errors.New("room not exist")
	}
	r.sendMessage(msg)
	return &room.Reply_SendMessage{}, nil
}
