package signal

import (
	"sync"

	"github.com/cloudwebrtc/go-protoo/room"
	"github.com/pion/ion/log"
)

var (
	rooms    = make(map[string]*Room)
	roomLock sync.RWMutex
)

type Room struct {
	room.Room
	// ID string
}

func (r *Room) AddPeer(peer *Peer) {
	r.Room.AddPeer(&peer.Peer)
}

func (r *Room) ID() string {
	return r.Room.ID()
}

func newRoom(id string) *Room {
	// r := &Room{ID: id}
	r := &Room{
		Room: *room.NewRoom(id),
	}
	// r.Room = *room.NewRoom(id)
	roomLock.Lock()
	rooms[id] = r
	roomLock.Unlock()
	return r
}

func getRoom(id string) *Room {
	roomLock.RLock()
	r := rooms[id]
	roomLock.RUnlock()
	log.Infof("GetRoom r=%+v", r)
	return r
}

func deleteRoom(id string) {
	roomLock.Lock()
	if rooms[id] != nil {
		rooms[id].Close()
	}
	delete(rooms, id)
	roomLock.Unlock()
}

func GetRoomPeerTotal(rid string) int {
	log.Infof("GetRoomPeerTotal rid=%s", rid)
	room := getRoom(rid)
	if room != nil {
		log.Infof("GetRoomPeerTotal len=%d", len(room.GetPeers()))
		return len(room.GetPeers())
	}
	return 0
}

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

func GetRoomsByPeer(id string) []*Room {
	var r []*Room
	roomLock.RLock()
	defer roomLock.RUnlock()
	for _, room := range rooms {
		if room == nil {
			continue
		}
		if peer := room.GetPeer(id); peer != nil {
			r = append(r, room)
		}
	}
	return r
}

func DeletePeer(id string) {
	log.Infof("DeletePeer id=%s", id)
	room := GetRoomByPeer(id)
	if room != nil {
		room.RemovePeer(id)
	}
}

func DeletePeerFromRoom(rid, id string) {
	log.Infof("DeletePeerFromRoom rid=%s id=%s", rid, id)
	room := getRoom(rid)
	if room != nil {
		room.RemovePeer(id)
	}
}

func AddPeerToRoom(rid string, peer *Peer) {
	log.Infof("AddPeerToRoom rid=%s peer.ID=%s", rid, peer.ID())
	room := getRoom(rid)
	if room == nil {
		room = newRoom(rid)
	}
	room.AddPeer(peer)
}

func NotifyAllWithoutPeer(rid string, peer *Peer, method string, msg map[string]interface{}) {
	room := getRoom(rid)
	if room != nil {
		log.Infof("room %s Notify method=%s msg=%v", rid, method, msg)
		room.Notify(&peer.Peer, method, msg)
	}
}

func NotifyAllWithoutID(rid string, skipID string, method string, msg map[string]interface{}) {
	room := getRoom(rid)
	if room != nil {
		peer := getPeer(rid, skipID)
		if peer == nil {
			log.Errorf("NotifyAllWithoutID peer == nil rid=%s  skipID=%s", rid, skipID)
			return
		}
		room.Notify(peer, method, msg)
	}
}

// if id != skipID notify to id
func NotifyByID(rid, id, skipID string, method string, msg map[string]interface{}) {
	room := getRoom(rid)
	if room == nil {
		return
	}
	peer := room.GetPeer(id)
	if peer != nil && peer.ID() != skipID {
		peer.Notify(method, msg)
	}
}
