package biz

import (
	"sync"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
)

var (
	roomLock sync.RWMutex
	rooms    = make(map[proto.RID]*room)
)

// room represents a room which manage peers
type room struct {
	sync.RWMutex
	id    proto.RID
	peers map[proto.UID]*peer
}

// newRoom creates a new room instance
func newRoom(id proto.RID) *room {
	r := &room{
		id:    id,
		peers: make(map[proto.UID]*peer),
	}

	roomLock.Lock()
	defer roomLock.Unlock()
	rooms[id] = r

	return r
}

// ID room id
func (r *room) ID() proto.RID {
	return r.id
}

// addPeer add a peer to room
func (r *room) addPeer(p *peer) {
	r.Lock()
	defer r.Unlock()
	r.peers[p.id] = p
}

// getPeer get a peer by peer id
func (r *room) getPeer(uid proto.UID) *peer {
	r.RLock()
	defer r.RUnlock()
	return r.peers[uid]
}

// getPeers get peers in the room
func (r *room) getPeers() map[proto.UID]*peer {
	r.RLock()
	defer r.RUnlock()
	return r.peers
}

// delPeer delete a peer in the room
func (r *room) delPeer(uid proto.UID) {
	r.Lock()
	defer r.Unlock()
	delete(r.peers, uid)
}

// notify notify a message to peers without one peer
func (r *room) notify(method string, data interface{}) {
	peers := r.getPeers()
	for _, p := range peers {
		p.notify(method, data)
	}
}

// notifyWithoutID notify a message to peers without one peer
func (r *room) notifyWithoutID(method string, data interface{}, withoutID proto.UID) {
	peers := r.getPeers()
	for id, p := range peers {
		if id != withoutID {
			p.notify(method, data)
		}
	}
}

// getRoom get a room by id
func getRoom(id proto.RID) *room {
	roomLock.RLock()
	defer roomLock.RUnlock()
	r := rooms[id]
	return r
}

// getRoomsByPeer a peer in many room
func getRoomsByPeer(uid proto.UID) []*room {
	var result []*room
	roomLock.RLock()
	defer roomLock.RUnlock()
	for _, r := range rooms {
		if p := r.getPeer(uid); p != nil {
			result = append(result, r)
		}
	}
	return result
}

// delPeer delete a peer in the room
func delPeer(rid proto.RID, uid proto.UID) {
	log.Infof("AddPeer rid=%s uid=%s", rid, uid)
	room := getRoom(rid)
	if room != nil {
		room.delPeer(uid)
	}
}

// addPeer add a peer to room
func addPeer(rid proto.RID, peer *peer) {
	log.Infof("AddPeer rid=%s uid=%s", rid, peer.id)
	room := getRoom(rid)
	if room == nil {
		room = newRoom(rid)
	}
	room.addPeer(peer)
}

// getPeer get a peer in the room
func getPeer(rid proto.RID, uid proto.UID) *peer {
	log.Infof("GetPeer rid=%s uid=%s", rid, uid)
	r := getRoom(rid)
	if r == nil {
		//log.Infof("room not exits, rid=%s uid=%s", rid, uid)
		return nil
	}
	return r.getPeer(uid)
}

// notifyPeer send message to peer
func notifyPeer(method string, rid proto.RID, uid proto.UID, data interface{}) {
	log.Infof("Notify rid=%s, uid=%s, data=%s", rid, uid, data)
	room := getRoom(rid)
	if room == nil {
		log.Errorf("room not exits, rid=%s, uid=%s, data=%s", rid, uid)
		return
	}
	peer := room.getPeer(uid)
	if peer == nil {
		log.Errorf("peer not exits, rid=%s, uid=%s, data=%s", rid, uid)
		return
	}
	peer.notify(method, data)
}
