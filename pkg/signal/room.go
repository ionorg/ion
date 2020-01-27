package signal

import (
	"github.com/cloudwebrtc/go-protoo/room"
	"github.com/pion/ion/pkg/log"
)

type Room struct {
	room.Room
}

func (r *Room) AddPeer(peer *Peer) {
	r.Room.AddPeer(&peer.Peer)
}

func (r *Room) ID() string {
	return r.Room.ID()
}

func newRoom(id string) *Room {
	r := &Room{
		Room: *room.NewRoom(id),
	}
	roomLock.Lock()
	rooms[id] = r
	roomLock.Unlock()
	return r
}

func getRoom(id string) *Room {
	roomLock.RLock()
	r := rooms[id]
	roomLock.RUnlock()
	log.Debugf("getRoom %v", r)
	return r
}

func delRoom(id string) {
	roomLock.Lock()
	if rooms[id] != nil {
		rooms[id].Close()
	}
	delete(rooms, id)
	roomLock.Unlock()
}

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
	for rid, room := range rooms {
		log.Infof("signal.GetRoomsByPeer rid=%v room=%v id=%v", rid, room, id)
		if room == nil {
			continue
		}
		if peer := room.GetPeer(id); peer != nil {
			r = append(r, room)
		}
	}
	return r
}

func DelPeer(rid, id string) {
	log.Debugf("DelPeer rid=%s id=%s", rid, id)
	room := getRoom(rid)
	if room != nil {
		room.RemovePeer(id)
	}
}

func AddPeer(rid string, peer *Peer) {
	log.Debugf("AddPeer rid=%s peer.ID=%s", rid, peer.ID())
	room := getRoom(rid)
	if room == nil {
		room = newRoom(rid)
	}
	room.AddPeer(peer)
}

func HasPeer(rid string, peer *Peer) bool {
	log.Debugf("HasPeer rid=%s peer.ID=%s", rid, peer.ID())
	room := getRoom(rid)
	if room == nil {
		return false
	}
	return room.GetPeer(peer.ID()) != nil
}

func NotifyAllWithoutPeer(rid string, peer *Peer, method string, msg map[string]interface{}) {
	log.Debugf("signal.NotifyAllWithoutPeer rid=%s peer.ID=%s method=%s msg=%v", rid, peer.ID(), method, msg)
	room := getRoom(rid)
	if room != nil {
		log.Debugf("room %s Notify method=%s msg=%v", rid, method, msg)
		room.Notify(&peer.Peer, method, msg)
	}
}

func NotifyAll(rid string, method string, msg map[string]interface{}) {
	room := getRoom(rid)
	if room != nil {
		for _, peer := range room.GetPeers() {
			if peer != nil {
				peer.Notify(method, msg)
			}
		}
	}
}

func NotifyAllWithoutID(rid string, skipID string, method string, msg map[string]interface{}) {
	room := getRoom(rid)
	if room != nil {
		for _, peer := range room.GetPeers() {
			if peer != nil && peer.ID() != skipID {
				peer.Notify(method, msg)
			}
		}
	}
}
