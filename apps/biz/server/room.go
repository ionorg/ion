package server

import (
	"sync"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/proto/ion"
)

// Room represents a Room which manage peers
type Room struct {
	sync.RWMutex
	sid   string
	nid   string
	peers map[string]*Peer
}

// newRoom creates a new room instance
func newRoom(sid string, nid string) *Room {
	r := &Room{
		sid:   sid,
		nid:   nid,
		peers: make(map[string]*Peer),
	}
	return r
}

// SID room id
func (r *Room) SID() string {
	return r.sid
}

// addPeer add a peer to room
func (r *Room) addPeer(p *Peer) {

	event := &ion.PeerEvent{
		State: ion.PeerEvent_JOIN,
		Peer: &ion.Peer{
			Sid:  r.sid,
			Uid:  p.uid,
			Info: p.info,
		},
	}
	r.sendPeerEvent(event)

	// Send the peer info in the existing room
	// to the newly added peer.
	for _, peer := range r.getPeers() {
		event := &ion.PeerEvent{
			State: ion.PeerEvent_JOIN,
			Peer: &ion.Peer{
				Sid:  r.sid,
				Uid:  peer.uid,
				Info: peer.info,
			},
		}
		err := p.sendPeerEvent(event)
		if err != nil {
			log.Errorf("p.sendPeerEvent() failed %v", err)
		}

		if peer.lastStreamEvent != nil {
			err := p.sendStreamEvent(peer.lastStreamEvent)
			if err != nil {
				log.Errorf("p.sendStreamEvent() failed %v", err)
			}
		}
	}

	r.Lock()
	r.peers[p.uid] = p
	r.Unlock()
}

// getPeer get a peer by peer id
func (r *Room) getPeer(uid string) *Peer {
	r.RLock()
	defer r.RUnlock()
	return r.peers[uid]
}

// getPeers get peers in the room
func (r *Room) getPeers() []*Peer {
	r.RLock()
	defer r.RUnlock()
	p := make([]*Peer, 0, len(r.peers))
	for _, peer := range r.peers {
		p = append(p, peer)
	}
	return p
}

// delPeer delete a peer in the room
func (r *Room) delPeer(p *Peer) int {
	uid := p.uid
	r.Lock()
	found := r.peers[uid] == p
	if found {
		delete(r.peers, uid)
	}
	peerCount := len(r.peers)
	r.Unlock()

	if found {
		event := &ion.PeerEvent{
			State: ion.PeerEvent_LEAVE,
			Peer: &ion.Peer{
				Sid: r.sid,
				Uid: uid,
			},
		}
		r.sendPeerEvent(event)
	}

	return peerCount
}

// count return count of peers in room
func (r *Room) count() int {
	r.RLock()
	defer r.RUnlock()
	return len(r.peers)
}

func (r *Room) sendPeerEvent(event *ion.PeerEvent) {
	peers := r.getPeers()
	for _, p := range peers {
		if err := p.sendPeerEvent(event); err != nil {
			log.Errorf("send data to peer(%s) error: %v", p.uid, err)
		}
	}
}

func (r *Room) sendStreamEvent(event *ion.StreamEvent) {
	peers := r.getPeers()
	for _, p := range peers {
		if err := p.sendStreamEvent(event); err != nil {
			log.Errorf("send data to peer(%s) error: %v", p.uid, err)
		}
	}
}

func (r *Room) sendMessage(msg *ion.Message) {
	from := msg.From
	to := msg.To
	data := msg.Data
	log.Debugf("Room.onMessage %v => %v, data: %v", from, to, data)
	peers := r.getPeers()
	for _, p := range peers {
		if to == p.uid || to == "all" || to == r.sid {
			if err := p.sendMessage(msg); err != nil {
				log.Errorf("send msg to peer(%s) error: %v", p.uid, err)
			}
		}
	}
}
