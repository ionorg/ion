package signal

import (
	"sync"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
)

// Room represents a room which manage peers
type Room struct {
	sync.RWMutex
	id    proto.RID
	peers map[proto.UID]*Peer
}

// newRoom creates a new room instance
func newRoom(id proto.RID) *Room {
	r := &Room{
		id:    id,
		peers: make(map[proto.UID]*Peer),
	}

	roomLock.Lock()
	defer roomLock.Unlock()
	rooms[id] = r

	return r
}

// ID room id
func (r *Room) ID() proto.RID {
	return r.id
}

// AddPeer add a peer to room
func (r *Room) AddPeer(p *Peer) {
	r.Lock()
	defer r.Unlock()
	r.peers[p.ID()] = p
}

// GetPeer get a peer by peer id
func (r *Room) GetPeer(uid proto.UID) *Peer {
	r.RLock()
	defer r.RUnlock()
	return r.peers[uid]
}

// GetPeers get peers in the room
func (r *Room) GetPeers() map[proto.UID]*Peer {
	r.RLock()
	defer r.RUnlock()
	return r.peers
}

// DelPeer delete a peer in the room
func (r *Room) DelPeer(uid proto.UID) {
	r.Lock()
	defer r.Unlock()
	delete(r.peers, uid)
}

// Notify notify a message to peers without one peer
func (r *Room) Notify(method string, data interface{}) {
	peers := r.GetPeers()
	for _, p := range peers {
		p.Notify(method, data)
	}
}

// NotifyWithoutID notify a message to peers without one peer
func (r *Room) NotifyWithoutID(method string, data interface{}, withoutID proto.UID) {
	peers := r.GetPeers()
	for id, p := range peers {
		if id != withoutID {
			p.Notify(method, data)
		}
	}
}

// GetRoom get a room by id
func GetRoom(id proto.RID) *Room {
	roomLock.RLock()
	defer roomLock.RUnlock()
	r := rooms[id]
	return r
}

// GetRoomsByPeer a peer in many room
func GetRoomsByPeer(uid proto.UID) []*Room {
	var result []*Room
	roomLock.RLock()
	defer roomLock.RUnlock()
	for _, r := range rooms {
		if p := r.GetPeer(uid); p != nil {
			result = append(result, r)
		}
	}
	return result
}

// DelPeer delete a peer in the room
func DelPeer(rid proto.RID, uid proto.UID) {
	log.Infof("AddPeer rid=%s uid=%s", rid, uid)
	room := GetRoom(rid)
	if room != nil {
		room.DelPeer(uid)
	}
}

// AddPeer add a peer to room
func AddPeer(rid proto.RID, peer *Peer) {
	log.Infof("AddPeer rid=%s uid=%s", rid, peer.ID())
	room := GetRoom(rid)
	if room == nil {
		room = newRoom(rid)
	}
	room.AddPeer(peer)
}

// GetPeer get a peer in the room
func GetPeer(rid proto.RID, uid proto.UID) *Peer {
	log.Infof("GetPeer rid=%s uid=%s", rid, uid)
	r := GetRoom(rid)
	if r == nil {
		//log.Infof("room not exits, rid=%s uid=%s", rid, uid)
		return nil
	}
	return r.GetPeer(uid)
}

// NotifyPeer send message to peer
func NotifyPeer(method string, rid proto.RID, uid proto.UID, data interface{}) {
	log.Infof("Notify rid=%s, uid=%s, data=%s", rid, uid, data)
	room := GetRoom(rid)
	if room == nil {
		log.Errorf("room not exits, rid=%s, uid=%s, data=%s", rid, uid)
		return
	}
	peer := room.GetPeer(uid)
	if peer == nil {
		log.Errorf("peer not exits, rid=%s, uid=%s, data=%s", rid, uid)
		return
	}
	peer.Notify(method, data)
}
