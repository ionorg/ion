package biz

import (
	"sync"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
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
	r.peers[p.uid] = p
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
func (r *room) delPeer(uid proto.UID) int {
	r.Lock()
	defer r.Unlock()
	delete(r.peers, uid)
	return len(r.peers)
}

// count return count of peers in room
func (r *room) count() int {
	r.RLock()
	defer r.RUnlock()
	return len(r.peers)
}

// // notify notify a message to peers without one peer
// func (r *room) notify(method string, data interface{}) {
// 	peers := r.getPeers()
// 	for _, p := range peers {
// 		if err := p.notify(method, data); err != nil {
// 			log.Errorf("send data to peer(%s) error: %v", p.uid, err)
// 		}
// 	}
// }

// notifyWithoutID notify a message to peers without one peer
func (r *room) notifyWithoutID(method string, data interface{}, withoutID proto.UID) {
	peers := r.getPeers()
	for id, p := range peers {
		if id != withoutID {
			if err := p.notify(method, data); err != nil {
				log.Errorf("send data to peer(%s) error: %v", p.uid, err)
			}
		}
	}
}
