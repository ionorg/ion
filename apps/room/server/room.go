package server

import (
	"sync"

	log "github.com/pion/ion-log"
	room "github.com/pion/ion/apps/room/proto"
	"github.com/pion/ion/proto/ion"
)

type global struct {
	Pprof string `mapstructure:"pprof"`
	Dc    string `mapstructure:"dc"`
}

type logConf struct {
	Level string `mapstructure:"level"`
}

type natsConf struct {
	URL string `mapstructure:"url"`
}

type nodeConf struct {
	NID string `mapstructure:"nid"`
}

// Config for biz node
type Config struct {
	Global global   `mapstructure:"global"`
	Log    logConf  `mapstructure:"log"`
	Nats   natsConf `mapstructure:"nats"`
	Node   nodeConf `mapstructure:"node"`
}

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
	event := &room.ParticipantEvent{
		Participant: &room.Participant{
			Uid:           p.uid,
			DisplayName:   "",                 //TODO
			ExtraInfo:     nil,                //TODO
			Role:          room.Role_RoleHost, //TODO
			Protocol:      room.Protocol_ProtocolWebRTC,
			Avatar:        "", //TODO
			CallDirection: "", //TODO
			Vendor:        "", //TODO
		},
		Status: room.ParticipantStatus_Create,
	}

	r.sendPeerEvent(event)

	// Send the peer info in the existing room
	// to the newly added peer.
	for _, peer := range r.getPeers() {
		event := &room.ParticipantEvent{
			Participant: &room.Participant{
				Uid:           peer.uid,
				DisplayName:   "",                 //TODO
				ExtraInfo:     peer.info,          //TODO
				Role:          room.Role_RoleHost, //TODO
				Protocol:      room.Protocol_ProtocolWebRTC,
				Avatar:        "", //TODO
				CallDirection: "", //TODO
				Vendor:        "", //TODO
			},
			Status: room.ParticipantStatus_Create,
		}
		err := p.sendPeerEvent(event)
		if err != nil {
			log.Errorf("p.sendPeerEvent() failed %v", err)
		}

		if peer.lastStreamEvent != nil {
			//err := p.sendStreamEvent(peer.lastStreamEvent)
			//if err != nil {
			//	log.Errorf("p.sendStreamEvent() failed %v", err)
			//}
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
		event := &room.ParticipantEvent{
			Participant: &room.Participant{
				Uid:           uid,
				DisplayName:   "",                 //TODO
				ExtraInfo:     nil,                //TODO
				Role:          room.Role_RoleHost, //TODO
				Protocol:      room.Protocol_ProtocolWebRTC,
				Avatar:        "", //TODO
				CallDirection: "", //TODO
				Vendor:        "", //TODO
			},
			Status: room.ParticipantStatus_Delete,
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

func (r *Room) sendPeerEvent(event *room.ParticipantEvent) {
	peers := r.getPeers()
	for _, p := range peers {
		if err := p.sendPeerEvent(event); err != nil {
			log.Errorf("send data to peer(%s) error: %v", p.uid, err)
		}
	}
}

func (r *Room) sendStreamEvent(event *ion.StreamEvent) {
	//peers := r.getPeers()
	//for _, p := range peers {
	//if err := p.sendStreamEvent(event); err != nil {
	//	log.Errorf("send data to peer(%s) error: %v", p.uid, err)
	//}
	//}
}

func (r *Room) sendMessage(msg *room.Message) {
	from := msg.Origin
	// to := msg.To //TODO
	to := ""
	data := msg.Payload
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
