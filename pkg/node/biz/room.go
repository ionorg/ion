package biz

import (
	"sync"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
)

// room represents a room which manage peers
type room struct {
	sync.RWMutex
	id    proto.SID
	peers map[proto.UID]*Peer
}

// newRoom creates a new room instance
func newRoom(id proto.SID) *room {
	r := &room{
		id:    id,
		peers: make(map[proto.UID]*Peer),
	}
	return r
}

// ID room id
func (r *room) ID() proto.SID {
	return r.id
}

// addPeer add a peer to room
func (r *room) addPeer(p *Peer) {
	r.Lock()
	defer r.Unlock()
	r.peers[p.uid] = p
}

// getPeer get a peer by peer id
func (r *room) getPeer(uid proto.UID) *Peer {
	r.RLock()
	defer r.RUnlock()
	return r.peers[uid]
}

// getPeers get peers in the room
func (r *room) getPeers() map[proto.UID]*Peer {
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

// send message to peers
func (r *room) send(data interface{}, without proto.UID) {
	peers := r.getPeers()
	for id, p := range peers {
		if len(without) > 0 && id != without {
			if err := p.send(data); err != nil {
				log.Errorf("send data to peer(%s) error: %v", p.uid, err)
			}
		}
	}
}
