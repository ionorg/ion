package server

import (
	"os"
	"sync"
	"time"

	log "github.com/pion/ion-log"
	room "github.com/pion/ion/apps/room/proto"
	"github.com/pion/ion/pkg/db"
	"github.com/spf13/viper"
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
	Global global    `mapstructure:"global"`
	Log    logConf   `mapstructure:"log"`
	Nats   natsConf  `mapstructure:"nats"`
	Node   nodeConf  `mapstructure:"node"`
	Redis  db.Config `mapstructure:"redis"`
}

func unmarshal(rawVal interface{}) error {
	if err := viper.Unmarshal(rawVal); err != nil {
		return err
	}
	return nil
}

func (c *Config) Load(file string) error {
	_, err := os.Stat(file)
	if err != nil {
		return err
	}

	viper.SetConfigFile(file)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		log.Errorf("config file %s read failed. %v\n", file, err)
		return err
	}

	err = unmarshal(c)
	if err != nil {
		return err
	}

	if err != nil {
		log.Errorf("config file %s loaded failed. %v\n", file, err)
		return err
	}

	log.Infof("config %s load ok!", file)
	return nil
}

// Room represents a Room which manage peers
type Room struct {
	sync.RWMutex
	sid    string
	peers  map[string]*Peer
	info   room.Room
	update time.Time
}

// newRoom creates a new room instance
func newRoom(sid string) *Room {
	r := &Room{
		sid:    sid,
		peers:  make(map[string]*Peer),
		update: time.Now(),
	}
	return r
}

// Room name
func (r *Room) Name() string {
	return r.info.Name
}

// SID room id
func (r *Room) SID() string {
	r.update = time.Now()
	return r.sid
}

// addPeer add a peer to room
func (r *Room) addPeer(p *Peer) {
	event := &room.PeerEvent{
		Peer:  &p.info,
		State: room.PeerState_JOIN,
	}

	r.broadcastPeerEvent(event)

	r.Lock()
	p.room = r
	r.peers[p.info.Uid] = p
	r.update = time.Now()
	r.Unlock()
}

func (r *Room) roomLocked() bool {
	r.RLock()
	defer r.RUnlock()
	r.update = time.Now()
	return r.info.Lock
}

// getPeer get a peer by peer id
func (r *Room) getPeer(uid string) *Peer {
	r.RLock()
	defer r.RUnlock()
	r.update = time.Now()
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
	uid := p.info.Uid
	r.Lock()
	r.update = time.Now()
	found := r.peers[uid] == p
	if !found {
		r.Unlock()
		return -1
	}

	delete(r.peers, uid)
	peerCount := len(r.peers)
	r.Unlock()

	event := &room.PeerEvent{
		Peer:  &p.info,
		State: room.PeerState_LEAVE,
	}
	r.broadcastPeerEvent(event)

	return peerCount
}

// count return count of peers in room
func (r *Room) count() int {
	r.RLock()
	defer r.RUnlock()
	r.update = time.Now()
	return len(r.peers)
}

func (r *Room) broadcastRoomEvent(event *room.Reply) {
	log.Infof("event=%+v", event)
	peers := r.getPeers()
	r.update = time.Now()
	for _, p := range peers {
		if err := p.send(event); err != nil {
			log.Errorf("send data to peer(%s) error: %v", p.info.Uid, err)
		}
	}
}

func (r *Room) broadcastPeerEvent(event *room.PeerEvent) {
	log.Infof("event=%+v", event)
	peers := r.getPeers()
	r.update = time.Now()
	for _, p := range peers {
		if p.info.Uid == event.Peer.Uid {
			continue
		}
		if err := p.sendPeerEvent(event); err != nil {
			log.Errorf("send data to peer(%s) error: %v", p.info.Uid, err)
		}
	}
}

func (r *Room) sendMessage(msg *room.Message) {
	log.Infof("msg=%+v", msg)
	r.update = time.Now()
	from := msg.From
	to := msg.To
	dtype := msg.Type
	data := msg.Payload
	log.Debugf("Room.onMessage %v => %v, type: %v, data: %v", from, to, dtype, data)
	peers := r.getPeers()
	for _, p := range peers {
		if to == p.info.Uid || to == "all" || to == r.sid {
			if err := p.sendMessage(msg); err != nil {
				log.Errorf("send msg to peer(%s) error: %v", p.info.Uid, err)
			}
		}
	}
}
