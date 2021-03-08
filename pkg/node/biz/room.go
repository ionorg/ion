package biz

import (
	"sync"

	log "github.com/pion/ion-log"
)

// Room represents a Room which manage peers
type Room struct {
	sync.RWMutex
	id    string
	nid   string
	peers map[string]*Peer
}

// newRoom creates a new room instance
func newRoom(id string, nid string) *Room {
	r := &Room{
		id:    id,
		nid:   nid,
		peers: make(map[string]*Peer),
	}
	return r
}

// ID room id
func (r *Room) ID() string {
	return r.id
}

// NID ID for sfu node.
func (r *Room) NID() string {
	return r.nid
}

// addPeer add a peer to room
func (r *Room) addPeer(p *Peer) {
	r.Lock()
	defer r.Unlock()
	r.peers[p.uid] = p
}

// getPeer get a peer by peer id
func (r *Room) getPeer(uid string) *Peer {
	r.RLock()
	defer r.RUnlock()
	return r.peers[uid]
}

// getPeers get peers in the room
func (r *Room) getPeers() map[string]*Peer {
	r.RLock()
	defer r.RUnlock()
	return r.peers
}

// delPeer delete a peer in the room
func (r *Room) delPeer(uid string) int {
	r.Lock()
	defer r.Unlock()
	delete(r.peers, uid)
	return len(r.peers)
}

// count return count of peers in room
func (r *Room) count() int {
	r.RLock()
	defer r.RUnlock()
	return len(r.peers)
}

// send message to peers
func (r *Room) send(data interface{}, without string) {
	peers := r.getPeers()
	for id, p := range peers {
		if len(without) > 0 && id != without {
			if err := p.send(data); err != nil {
				log.Errorf("send data to peer(%s) error: %v", p.uid, err)
			}
		}
	}
}
