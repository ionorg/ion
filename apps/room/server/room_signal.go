package server

import (
	"errors"
	"io"

	log "github.com/pion/ion-log"
	room "github.com/pion/ion/apps/room/proto"
	"github.com/pion/ion/pkg/util"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// RoomSignalService represents an RoomSignalService instance
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
			if err == io.EOF {
				return nil
			}

			errStatus, _ := status.FromError(err)
			if errStatus.Code() == codes.Canceled {
				return nil
			}

			log.Errorf("signal error %v %v", errStatus.Message(), errStatus.Code())
			return err
		}

		switch payload := req.Payload.(type) {
		case *room.Request_Join:
			log.Infof("[C->S]=%+v", payload)
			reply, peer, err := s.Join(payload, stream)
			if err != nil {
				log.Errorf("Join err: %v", err)
				return err
			}

			p = peer
			log.Infof("[S->C]=%+v", reply)
			err = stream.Send(&room.Reply{Payload: reply})
			if err != nil {
				log.Errorf("stream send error: %v", err)
			}
		case *room.Request_Leave:
			log.Infof("[C->S]=%+v", payload)
			reply, err := s.Leave(payload)
			if err != nil {
				log.Errorf("Leave err: %v", err)
				return err
			}
			log.Infof("[S->C]=%+v", reply)
			err = stream.Send(&room.Reply{Payload: reply})
			if err != nil {
				log.Errorf("stream send error: %v", err)
			}
		case *room.Request_SendMessage:
			log.Infof("[C->S]=%+v", payload)
			reply, err := s.SendMessage(payload)
			if err != nil {
				log.Errorf("LockConference err: %v", err)
				return err
			}
			log.Infof("[S->C]=%+v", reply)
			err = stream.Send(&room.Reply{Payload: reply})
			if err != nil {
				log.Errorf("stream send error: %v", err)
			}

		case *room.Request_UpdateRoom:
			log.Infof("[C->S]=%+v", payload)
			reply, err := s.UpdateRoom(payload)
			if err != nil {
				log.Errorf("UpdateRoom err: %v", err)
				return err
			}
			log.Infof("[S->C]=%+v", reply)
			if err != nil {
				log.Errorf("stream send error: %v", err)
			}
		default:
			log.Errorf("unknown signal!! payload=%+v", payload)
		}
	}
}

func (s *RoomSignalService) Join(in *room.Request_Join, stream room.RoomSignal_SignalServer) (*room.Reply_Join, *Peer, error) {
	pinfo := in.Join.Peer

	if pinfo == nil || pinfo.Sid == "" && pinfo.Uid == "" {
		reply := &room.Reply_Join{
			Join: &room.JoinReply{
				Success: false,
				Room:    nil,
				Error: &room.Error{
					Code:   room.ErrorType_InvalidParams,
					Reason: "sid/uid is empty",
				},
			},
		}
		return reply, nil, status.Errorf(codes.Internal, "sid/uid is empty")
	}

	sid := pinfo.Sid
	uid := pinfo.Uid

	key := util.GetRedisRoomKey(sid)
	// create in redis if room not exist
	if sid == "" {
		// store room info
		sid = pinfo.Sid
		err := s.rs.redis.HMSetTTL(roomRedisExpire, key, "sid", sid, "name", pinfo.DisplayName,
			"password", "", "description", "", "lock", "0")
		if err != nil {
			reply := &room.Reply_Join{
				Join: &room.JoinReply{
					Success: false,
					Room:    nil,
					Error: &room.Error{
						Code:   room.ErrorType_ServiceUnavailable,
						Reason: err.Error(),
					},
				},
			}
			return reply, nil, err
		}
	}

	var peer *Peer = nil
	r := s.rs.getRoom(sid)

	if r == nil {
		// create room and store
		r = s.rs.createRoom(sid)
		err := s.rs.redis.HMSetTTL(roomRedisExpire, key, "sid", sid, "name", "",
			"password", "", "description", "", "lock", false)
		if err != nil {
			reply := &room.Reply_Join{
				Join: &room.JoinReply{
					Success: false,
					Room:    nil,
					Error: &room.Error{
						Code:   room.ErrorType_ServiceUnavailable,
						Reason: err.Error(),
					},
				},
			}
			return reply, nil, err
		}
	}

	// some one didn't create room before join
	if r.info == nil {
		r.info = new(room.Room)
	}
	r.info.Sid = sid

	peer = NewPeer()
	peer.info = pinfo
	peer.sig = stream
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

	// store peer to redis
	key = util.GetRedisPeerKey(sid, uid)
	err := s.rs.redis.HMSetTTL(roomRedisExpire, key, "sid", sid, "uid", uid, "dest", in.Join.Peer.Destination,
		"name", in.Join.Peer.DisplayName, "role", in.Join.Peer.Role.String(), "protocol", in.Join.Peer.Protocol.String(), "direction", in.Join.Peer.Direction.String(), "avatar", in.Join.Peer.Avatar, "info", in.Join.Peer.ExtraInfo)
	if err != nil {
		reply := &room.Reply_Join{
			Join: &room.JoinReply{
				Success: false,
				Room:    nil,
				Error: &room.Error{
					Code:   room.ErrorType_ServiceUnavailable,
					Reason: err.Error(),
				},
			},
		}
		return reply, nil, err
	}

	reply := &room.Reply_Join{
		Join: &room.JoinReply{
			Success: true,
			Room:    r.info,
			Error:   nil,
		},
	}

	// find peer in room
	key = util.GetRedisPeersPrefixKey(sid)
	peersKeys := s.rs.redis.Keys(key)

	for _, pkey := range peersKeys {
		res := s.rs.redis.HGetAll(pkey)
		sid = res["sid"]
		uid = res["uid"]
		if sid == "" || uid == "" || uid == pinfo.Uid {
			continue
		}
		key = util.GetRedisPeerKey(sid, uid)
		res = s.rs.redis.HGetAll(key)
		if len(res) != 0 {
			info := &room.Peer{
				Sid:         res["sid"],
				Uid:         res["uid"],
				DisplayName: res["name"],
				ExtraInfo:   []byte(res["info"]),
				Role:        room.Role(room.Role_value[res["role"]]),
				Protocol:    room.Protocol(room.Protocol_value[res["protocol"]]),
				Avatar:      res["avatar"],
				Direction:   room.Peer_Direction(room.Peer_Direction_value["direction"]),
				Vendor:      res["vendor"],
			}

			err := peer.sendPeerEvent(&room.PeerEvent{
				State: room.PeerState_JOIN,
				Peer:  info,
			})

			if err != nil {
				log.Errorf("signal send peer event error: %v", err)
			}
		}
	}
	log.Infof("Join OK: replay=%+v", reply)
	return reply, peer, nil
}

func (s *RoomSignalService) Leave(in *room.Request_Leave) (*room.Reply_Leave, error) {
	sid := in.Leave.Sid
	uid := in.Leave.Uid
	if sid == "" || uid == "" {
		return &room.Reply_Leave{
			Leave: &room.LeaveReply{
				Success: false,
				Error: &room.Error{
					Code:   room.ErrorType_RoomNotExist,
					Reason: "sid/uid is empty",
				},
			},
		}, status.Errorf(codes.Internal, "sid/uid is empty")
	}

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
	if peer != nil && peer.info.Uid == uid {
		if r.delPeer(peer) == 0 {
			s.rs.delRoom(r)
			r = nil
		}
	}

	reply := &room.Reply_Leave{
		Leave: &room.LeaveReply{
			Success: true,
			Error:   nil,
		},
	}

	return reply, nil
}

func (s *RoomSignalService) SendMessage(in *room.Request_SendMessage) (*room.Reply_SendMessage, error) {
	msg := in.SendMessage.Message
	sid := in.SendMessage.Sid
	log.Infof("Message: %+v", msg)
	r := s.rs.getRoom(sid)
	if r == nil {
		log.Warnf("room not found, maybe the peer did not join")
		return &room.Reply_SendMessage{}, errors.New("room not exist")
	}
	r.sendMessage(msg)
	return &room.Reply_SendMessage{}, nil
}

func (s *RoomSignalService) UpdateRoom(in *room.Request_UpdateRoom) (*room.UpdateRoomReply, error) {
	// TODO: Do not allow update room for peers that are host
	reply, err := s.rs.UpdateRoom(nil, in.UpdateRoom)
	return reply, err
}
