package biz

import (
	"sync"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/grpc/ion"
)

// Room represents a Room which manage peers
type Room struct {
	sync.RWMutex
	sid    string
	sfuNID string
	peers  map[string]*Peer
}

// newRoom creates a new room instance
func newRoom(sid string, nid string) *Room {
	r := &Room{
		sid:    sid,
		sfuNID: nid,
		peers:  make(map[string]*Peer),
	}
	return r
}

// SID room id
func (r *Room) SID() string {
	return r.sid
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

func (r *Room) broadcastPeerEvent(event *ion.PeerEvent) {
	peers := r.getPeers()
	for _, p := range peers {
		if err := p.sendPeerEvent(event); err != nil {
			log.Errorf("send data to peer(%s) error: %v", p.uid, err)
		}
	}
}

func (r *Room) broadcastStreamEvent(event *ion.StreamEvent) {
	peers := r.getPeers()
	for _, p := range peers {
		if err := p.sendStreamEvent(event); err != nil {
			log.Errorf("send data to peer(%s) error: %v", p.uid, err)
		}
	}
}

func (r *Room) broadcastMessage(msg *ion.Message) {
	from := msg.From
	to := msg.To
	data := msg.Data
	log.Debugf("Room.onMessage %v => %v, data: %v", from, to, data)
	peers := r.getPeers()
	for id, p := range peers {
		if id == to || to == "all" {
			if err := p.sendMessage(msg); err != nil {
				log.Errorf("send msg to peer(%s) error: %v", p.uid, err)
			}
		}
	}
}
