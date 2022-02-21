package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/pion/ion-log"
	room "github.com/pion/ion/apps/room/proto"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/util"
)

var (
	roomExpire      = time.Second * 10
	roomRedisExpire = 24 * time.Hour
)

type RoomService struct {
	room.UnimplementedRoomServiceServer
	roomLock sync.RWMutex
	rooms    map[string]*Room
	closed   chan struct{}
	redis    *db.Redis
}

func NewRoomService(config db.Config) *RoomService {
	s := &RoomService{
		rooms:  make(map[string]*Room),
		closed: make(chan struct{}),
		redis:  db.NewRedis(config),
	}
	go s.stat()
	return s
}

func (s *RoomService) Close() {
	close(s.closed)
}

// CreateRoom create a room
func (s *RoomService) CreateRoom(ctx context.Context, in *room.CreateRoomRequest) (*room.CreateRoomReply, error) {
	info := in.Room
	log.Infof("info=%+v", info)
	if info == nil || info.Sid == "" {
		return &room.CreateRoomReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_InvalidParams,
				Reason: "sid not exist",
			},
		}, nil
	}

	key := util.GetRedisRoomKey(info.Sid)

	// create local room if room not found locally
	r := s.getRoom(info.Sid)
	if r == nil {
		r = s.createRoom(info.Sid)
	}

	r.info = info //copy mutex?

	// store room info
	err := s.redis.HMSetTTL(roomRedisExpire, key, "sid", r.info.Sid, "name", r.info.Name,
		"password", r.info.Password, "description", r.info.Description, "lock", r.info.Lock)
	if err != nil {
		return &room.CreateRoomReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_ServiceUnavailable,
				Reason: err.Error(),
			},
		}, nil
	}

	log.Infof("create room ok sid=%v err=%v", info.Sid, err)

	// success
	return &room.CreateRoomReply{Success: true}, err
}

// UpdateRoom update a room.
// can lock room and change password
func (s *RoomService) UpdateRoom(ctx context.Context, in *room.UpdateRoomRequest) (*room.UpdateRoomReply, error) {
	info := in.Room
	log.Infof("info=%+v", info)
	if info == nil || info.Sid == "" {
		return &room.UpdateRoomReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_InvalidParams,
				Reason: "sid not exist",
			},
		}, nil
	}

	// check room in redis
	key := util.GetRedisRoomKey(info.Sid)

	// NOT CHECK
	// if s.redis.HGet(key, "sid") == "" {
	// 	return &room.UpdateRoomReply{
	// 		Success: false,
	// 		Error: &room.Error{
	// 			Code:   room.ErrorType_RoomNotExist,
	// 			Reason: "room not exist",
	// 		},
	// 	}, nil
	// }

	// check local room
	r := s.getRoom(info.Sid)
	if r == nil {
		r = s.createRoom(info.Sid)
	}
	r.info = info
	// update redis
	log.Infof("update room info=%+v", r.info)
	err := s.redis.HMSetTTL(roomRedisExpire, key, "sid", r.info.Sid, "name", r.info.Name,
		"password", r.info.Password, "description", r.info.Description, "lock", r.info.Lock)
	if err != nil {
		return &room.UpdateRoomReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_ServiceUnavailable,
				Reason: err.Error(),
			},
		}, nil
	}

	event := &room.Reply{
		Payload: &room.Reply_Room{
			Room: info,
		},
	}

	// broadcast to others
	r.broadcastRoomEvent("", event)
	log.Infof("update room ok sid=%v", info.Sid)
	return &room.UpdateRoomReply{Success: true}, nil
}

// EndRoom end a room
func (s *RoomService) EndRoom(ctx context.Context, in *room.EndRoomRequest) (*room.EndRoomReply, error) {
	sid := in.Sid
	log.Infof("sid=%+v", sid)
	if sid == "" {
		return &room.EndRoomReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_InvalidParams,
				Reason: "sid not exist",
			},
		}, nil
	}

	// delete room in redis
	key := util.GetRedisRoomKey(sid)

	if in.Delete {
		err := s.redis.Del(key)
		if err != nil {
			return &room.EndRoomReply{
				Success: false,
				Error: &room.Error{
					Code:   room.ErrorType_ServiceUnavailable,
					Reason: err.Error(),
				},
			}, nil
		}
	}

	// broadcast end message if delete room in redis ok
	event := &room.Disconnect{
		Sid:    sid,
		Reason: "Room ended",
	}

	r := s.getRoom(sid)
	if r != nil {
		// delete all peers and room
		peers := r.getPeers()
		for _, p := range peers {
			if p.sig == nil {
				continue
			}
			p.sig.Context().Done()
			r.delPeer(p)
		}
		s.delRoom(r)
	}

	r.broadcastRoomEvent(
		"",
		&room.Reply{
			Payload: &room.Reply_Disconnect{
				Disconnect: event,
			},
		},
	)

	log.Infof("end room ok sid=%v", sid)
	return &room.EndRoomReply{Success: true}, nil
}

// AddPeer invite a peer (webrtc/rtsp/rtmp/.. stream)
func (s *RoomService) AddPeer(ctx context.Context, in *room.AddPeerRequest) (*room.AddPeerReply, error) {
	log.Infof("AddPeer in=%+v", in)
	info := in.Peer
	if info == nil || info.Uid == "" || info.Sid == "" {
		return &room.AddPeerReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_InvalidParams,
				Reason: "sid or uid not exist",
			},
		}, nil
	}

	sid := in.Peer.Sid
	uid := in.Peer.Uid
	dest := in.Peer.Destination
	name := in.Peer.DisplayName
	role := in.Peer.Role.String()
	protocol := in.Peer.Protocol.String()
	direction := in.Peer.Direction.String()
	avatar := in.Peer.Avatar
	extraInfo := in.Peer.ExtraInfo

	// check room exist
	key := util.GetRedisRoomKey(sid)
	if s.redis.HGet(key, "sid") == "" {
		return &room.AddPeerReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_RoomAlreadyExist,
				Reason: "room not exist",
			},
		}, nil
	}

	// check peer exist
	// key = util.GetRedisPeerKey(sid, uid)
	// if s.redis.HGet(key, "uid") != "" {
	// 	return &room.AddPeerReply{
	// 		Success: false,
	// 		Error: &room.Error{
	// 			Code:   room.ErrorType_PeerAlreadyExist,
	// 			Reason: "peer already exist",
	// 		},
	// 	}, nil
	// }

	// create room if not exist locally
	r := s.getRoom(sid)
	if r == nil {
		// recover room and info
		r = s.createRoom(sid)
		key = util.GetRedisRoomKey(sid)
		res := s.redis.HGetAll(key)
		r.info.Sid = res["sid"]
		r.info.Name = res["name"]
		r.info.Lock = util.StringToBool(res["lock"])
		r.info.Password = res["password"]
	}

	// create peer and add to room
	p := NewPeer()
	p.info = info
	r.addPeer(p)

	// store peer to redis
	key = util.GetRedisPeerKey(sid, uid)
	err := s.redis.HMSetTTL(roomRedisExpire, key, "sid", sid, "uid", uid, "dest", dest,
		"name", name, "role", role, "protocol", protocol, "direction", direction,
		"avatar", avatar, "info", extraInfo)

	if err != nil {
		return &room.AddPeerReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_ServiceUnavailable,
				Reason: err.Error(),
			},
		}, nil
	}
	log.Infof("add peer ok sid=%v", sid)
	return &room.AddPeerReply{Success: true}, nil
}

// UpdatePeer update a peer
func (s *RoomService) UpdatePeer(ctx context.Context, in *room.UpdatePeerRequest) (*room.UpdatePeerReply, error) {
	log.Infof("UpdatePeer in=%+v", in)
	info := in.Peer
	if info == nil || info.Uid == "" || info.Sid == "" {
		return &room.UpdatePeerReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_InvalidParams,
				Reason: "sid or uid not exist",
			},
		}, nil
	}
	sid := in.Peer.Sid
	uid := in.Peer.Uid
	destination := in.Peer.Destination
	name := in.Peer.DisplayName
	role := in.Peer.Role.String()
	protocol := in.Peer.Protocol.String()
	direction := in.Peer.Direction.String()
	avatar := in.Peer.Avatar
	extraInfo := in.Peer.ExtraInfo

	// check room exist
	key := util.GetRedisRoomKey(sid)
	if s.redis.HGet(key, "sid") == "" {
		return &room.UpdatePeerReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_RoomAlreadyExist,
				Reason: "room not exist",
			},
		}, nil
	}

	// check peer exist
	log.Infof("sid=%v  uid=%v", sid, uid)
	key = util.GetRedisPeerKey(sid, uid)
	if s.redis.HGet(key, "uid") == "" {
		return &room.UpdatePeerReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_PeerNotExist,
				Reason: "peer not exist",
			},
		}, nil
	}

	// create room if not exist locally
	r := s.getRoom(sid)
	if r == nil {
		// recover room and info
		r = s.createRoom(sid)
		key = util.GetRedisRoomKey(sid)
		res := s.redis.HGetAll(key)
		r.info.Sid = res["sid"]
		r.info.Name = res["name"]
		r.info.Lock = util.StringToBool(res["lock"])
		r.info.Password = res["password"]
	}

	// update local peer if exist
	p := r.getPeer(uid)
	if p == nil {
		p = NewPeer()
		r.addPeer(p)
	}
	p.info = info

	// store peer to redis
	key = util.GetRedisPeerKey(sid, uid)

	err := s.redis.HMSetTTL(roomRedisExpire, key, "sid", sid, "uid", uid, "destination", destination,
		"name", name, "role", role, "protocol", protocol, "direction", direction, "avatar", avatar, "info", extraInfo)

	if err != nil {
		return &room.UpdatePeerReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_ServiceUnavailable,
				Reason: err.Error(),
			},
		}, nil
	}

	// broadcast to others
	r.broadcastPeerEvent(
		&room.PeerEvent{
			Peer:  p.info,
			State: room.PeerState_UPDATE,
		},
	)
	log.Infof("update peer ok sid=%v", sid)
	return &room.UpdatePeerReply{Success: true}, nil
}

// RemovePeer delete a peer
func (s *RoomService) RemovePeer(ctx context.Context, in *room.RemovePeerRequest) (*room.RemovePeerReply, error) {
	sid := in.Sid
	uid := in.Uid

	// find room
	r := s.getRoom(sid)
	if r == nil {
		return &room.RemovePeerReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_RoomNotExist,
				Reason: "room not exist",
			},
		}, nil
	}

	// rm peer in db
	// store peer to redis
	key := util.GetRedisPeerKey(sid, uid)

	err := s.redis.Del(key)
	if err != nil {
		return &room.RemovePeerReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_ServiceUnavailable,
				Reason: err.Error(),
			},
		}, nil
	}

	// rm peer in room
	p := r.getPeer(uid)
	if p != nil {
		//remove peer
		_ = p.send(&room.Reply{
			Payload: &room.Reply_Disconnect{
				Disconnect: &room.Disconnect{
					Sid:    sid,
					Reason: "kicked by host",
				},
			},
		})
		r.delPeer(p)
	}
	log.Infof("remove peer ok sid=%v", sid)
	return &room.RemovePeerReply{Success: true}, nil
}

// GetPeers get all peers in room
func (s *RoomService) GetPeers(ctx context.Context, in *room.GetPeersRequest) (*room.GetPeersReply, error) {
	sid := in.Sid
	// get room and check
	r := s.getRoom(sid)
	if r == nil {
		return &room.GetPeersReply{
			Success: false,
			Error: &room.Error{
				Code:   room.ErrorType_RoomNotExist,
				Reason: "room not exist",
			},
		}, nil
	}

	// store peer to redis
	key := util.GetRedisPeersPrefixKey(sid)
	peersKeys := s.redis.Keys(key)
	var roomPeers []*room.Peer
	for _, pkey := range peersKeys {
		res := s.redis.HGetAll(pkey)
		roomPeer := &room.Peer{
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
		roomPeers = append(roomPeers, roomPeer)
	}
	log.Infof("get peers ok roomPeers=%+v", roomPeers)
	return &room.GetPeersReply{
		Success: true,
		Peers:   roomPeers,
	}, nil
}

func (s *RoomService) createRoom(sid string) *Room {
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	if r := s.rooms[sid]; r != nil {
		return r
	}
	r := newRoom(sid, s.redis)
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
	if s.rooms[id] != nil {
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
		s.roomLock.Lock()
		for sid, room := range s.rooms {
			//clean after room is clean and expired
			duration := time.Since(room.update)
			if duration > roomExpire && room.count() == 0 {
				s.roomLock.Lock()
				delete(s.rooms, sid)
				s.roomLock.Unlock()
			}
			info += fmt.Sprintf("room: %s\npeers: %d\n", sid, room.count())
		}
		s.roomLock.Unlock()
		if len(info) > 0 {
			log.Infof("\n----------------signal-----------------\n" + info)
		}
	}
}
