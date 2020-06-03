package signal

import (
	"github.com/cloudwebrtc/go-protoo/peer"
	"github.com/cloudwebrtc/go-protoo/room"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
)

type Room struct {
	room.Room
}

func (r *Room) AddPeer(peer *Peer) {
	r.Room.AddPeer(&peer.Peer)
}

func (r *Room) ID() proto.RID {
	return proto.RID(r.Room.ID())
}

func newRoom(id proto.RID) *Room {
	r := &Room{
		Room: *room.NewRoom(string(id)),
	}
	roomLock.Lock()
	rooms[id] = r
	roomLock.Unlock()
	return r
}

func getRoom(id proto.RID) *Room {
	roomLock.RLock()
	r := rooms[id]
	roomLock.RUnlock()
	log.Debugf("getRoom %v", r)
	return r
}

// func delRoom(id string) {
// 	roomLock.Lock()
// 	if rooms[id] != nil {
// 		rooms[id].Close()
// 	}
// 	delete(rooms, id)
// 	roomLock.Unlock()
// }

// one peer in one room
func GetRoomByPeer(id string) *Room {
	roomLock.RLock()
	defer roomLock.RUnlock()
	for _, room := range rooms {
		if room == nil {
			continue
		}
		if peer := room.GetPeer(id); peer != nil {
			return room
		}
	}
	return nil
}

// one peer in many room
func GetRoomsByPeer(id string) []*Room {
	var r []*Room
	roomLock.RLock()
	defer roomLock.RUnlock()
	for _, room := range rooms {
		//log.Debugf("signal.GetRoomsByPeer rid=%v id=%v", rid, id)
		if room == nil {
			continue
		}
		if peer := room.GetPeer(id); peer != nil {
			r = append(r, room)
		}
	}
	return r
}

func DelPeer(rid proto.RID, id string) {
	log.Infof("DelPeer rid=%s id=%s", rid, id)
	room := getRoom(rid)
	if room != nil {
		room.RemovePeer(id)
	}
}

func AddPeer(rid proto.RID, peer *Peer) {
	log.Infof("AddPeer rid=%s peer.ID=%s", rid, peer.ID())
	room := getRoom(rid)
	if room == nil {
		room = newRoom(rid)
	}
	room.AddPeer(peer)
}

func HasPeer(rid proto.RID, peer *Peer) bool {
	log.Debugf("HasPeer rid=%s peer.ID=%s", rid, peer.ID())
	room := getRoom(rid)
	if room == nil {
		return false
	}
	return room.GetPeer(peer.ID()) != nil
}

func NotifyAllWithoutPeer(rid proto.RID, peer *Peer, method string, msg interface{}) {
	log.Debugf("signal.NotifyAllWithoutPeer rid=%s peer.ID=%s method=%s msg=%v", rid, peer.ID(), method, msg)
	room := getRoom(rid)
	if room != nil {
		log.Debugf("room %s Notify method=%s msg=%v", rid, method, msg)
		room.Notify(&peer.Peer, method, msg)
	}
}

func NotifyAll(rid proto.RID, method string, msg interface{}) {
	room := getRoom(rid)
	if room != nil {
		room.Map(func(id string, peer *peer.Peer) {
			if peer != nil {
				peer.Notify(method, msg)
			}
		})
	}
}

func NotifyAllWithoutID(rid proto.RID, skipID proto.UID, method string, msg interface{}) {
	room := getRoom(rid)
	log.Infof("room => %v", rid)
	if room != nil {
		room.Map(func(id string, peer *peer.Peer) {
			if peer != nil && proto.UID(peer.ID()) != skipID {
				peer.Notify(method, msg)
			}
		})
	}
}
