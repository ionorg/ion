package server

import (
	"os"
	"sync"
	"time"

	natsDiscoveryClient "github.com/cloudwebrtc/nats-discovery/pkg/client"
	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	natsRPC "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/cloudwebrtc/nats-grpc/pkg/rpc/reflection"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"

	room "github.com/pion/ion/apps/room/proto"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/ion"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/runner"
	"github.com/pion/ion/pkg/util"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

type global struct {
	Dc string `mapstructure:"dc"`
}

type logConf struct {
	Level string `mapstructure:"level"`
}

type natsConf struct {
	URL string `mapstructure:"url"`
}

// Config for room node
type Config struct {
	runner.ConfigBase
	Global global    `mapstructure:"global"`
	Log    logConf   `mapstructure:"log"`
	Nats   natsConf  `mapstructure:"nats"`
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
	info   *room.Room
	update time.Time
	redis  *db.Redis
}

type RoomServer struct {
	// for standalone running
	runner.Service

	// grpc room service
	RoomService
	RoomSignalService

	// for distributed node running
	ion.Node
	natsConn         *nats.Conn
	natsDiscoveryCli *natsDiscoveryClient.Client

	// config
	conf Config
}

// New create a room node instance
func New() *RoomServer {
	return &RoomServer{
		Node: ion.NewNode("room-" + util.RandomString(6)),
	}
}

// Load load config file
func (r *RoomServer) Load(confFile string) error {
	err := r.conf.Load(confFile)
	if err != nil {
		log.Errorf("config load error: %v", err)
		return err
	}
	return nil
}

// ConfigBase used for runner
func (r *RoomServer) ConfigBase() runner.ConfigBase {
	return &r.conf
}

// StartGRPC for standalone bin
func (r *RoomServer) StartGRPC(registrar grpc.ServiceRegistrar) error {
	var err error

	ndc, err := natsDiscoveryClient.NewClient(nil)
	if err != nil {
		log.Errorf("failed to create discovery client: %v", err)
		ndc.Close()
		return err
	}

	r.natsDiscoveryCli = ndc
	r.natsConn = nil
	r.RoomService = *NewRoomService(r.conf.Redis)
	log.Infof("NewRoomService r.conf.Redis=%+v r.redis=%+v", r.conf.Redis, r.redis)
	r.RoomSignalService = *NewRoomSignalService(&r.RoomService)

	room.RegisterRoomServiceServer(registrar, &r.RoomService)
	room.RegisterRoomSignalServer(registrar, &r.RoomSignalService)

	return nil
}

// Start for distributed node
func (r *RoomServer) Start() error {
	var err error

	log.Infof("r.conf.Nats.URL===%+v", r.conf.Nats.URL)
	err = r.Node.Start(r.conf.Nats.URL)
	if err != nil {
		r.Close()
		return err
	}

	ndc, err := natsDiscoveryClient.NewClient(r.NatsConn())
	if err != nil {
		log.Errorf("failed to create discovery client: %v", err)
		ndc.Close()
		return err
	}

	r.natsDiscoveryCli = ndc
	r.natsConn = r.NatsConn()
	r.RoomService = *NewRoomService(r.conf.Redis)
	log.Infof("NewRoomService r.conf.Redis=%+v r.redis=%+v", r.conf.Redis, r.redis)
	r.RoomSignalService = *NewRoomSignalService(&r.RoomService)

	if err != nil {
		r.Close()
		return err
	}

	room.RegisterRoomServiceServer(r.Node.ServiceRegistrar(), &r.RoomService)
	room.RegisterRoomSignalServer(r.Node.ServiceRegistrar(), &r.RoomSignalService)
	// Register reflection service on nats-rpc server.
	reflection.Register(r.Node.ServiceRegistrar().(*natsRPC.Server))

	node := discovery.Node{
		DC:      r.conf.Global.Dc,
		Service: proto.ServiceROOM,
		NID:     r.Node.NID,
		RPC: discovery.RPC{
			Protocol: discovery.NGRPC,
			Addr:     r.conf.Nats.URL,
		},
	}

	go func() {
		err := r.Node.KeepAlive(node)
		if err != nil {
			log.Errorf("Room.Node.KeepAlive(%v) error %v", r.Node.NID, err)
		}
	}()

	//Watch ALL nodes.
	go func() {
		err := r.Node.Watch(proto.ServiceALL)
		if err != nil {
			log.Errorf("Node.Watch(proto.ServiceALL) error %v", err)
		}
	}()
	return nil
}

func (s *RoomServer) Close() {
	s.RoomService.Close()
	s.Node.Close()
}

// newRoom creates a new room instance
func newRoom(sid string, redis *db.Redis) *Room {
	r := &Room{
		sid:    sid,
		peers:  make(map[string]*Peer),
		update: time.Now(),
		redis:  redis,
	}
	return r
}

// Room name
func (r *Room) Name() string {
	return r.info.Name
}

// SID room id
func (r *Room) SID() string {
	return r.sid
}

// addPeer add a peer to room
func (r *Room) addPeer(p *Peer) {
	event := &room.PeerEvent{
		Peer:  p.info,
		State: room.PeerState_JOIN,
	}

	r.broadcastPeerEvent(event)

	r.Lock()
	p.room = r
	r.peers[p.info.Uid] = p
	r.update = time.Now()
	r.Unlock()
}

// func (r *Room) roomLocked() bool {
// 	r.RLock()
// 	defer r.RUnlock()
// 	r.update = time.Now()
// 	return r.info.Lock
// }

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
		Peer:  p.info,
		State: room.PeerState_LEAVE,
	}

	key := util.GetRedisPeerKey(p.info.Sid, uid)
	err := r.redis.Del(key)
	if err != nil {
		log.Errorf("err=%v", err)
	}

	r.broadcastPeerEvent(event)

	return peerCount
}

// count return count of peers in room
func (r *Room) count() int {
	r.RLock()
	defer r.RUnlock()
	return len(r.peers)
}

func (r *Room) broadcastRoomEvent(uid string, event *room.Reply) {
	log.Infof("event=%+v", event)
	peers := r.getPeers()
	r.update = time.Now()
	for _, p := range peers {
		if p.UID() == uid {
			continue
		}

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
	if to == "all" {
		r.broadcastRoomEvent(
			from,
			&room.Reply{
				Payload: &room.Reply_Message{
					Message: msg,
				},
			},
		)
		return
	}

	peers := r.getPeers()
	for _, p := range peers {
		if to == p.info.Uid {
			if err := p.sendMessage(msg); err != nil {
				log.Errorf("send msg to peer(%s) error: %v", p.info.Uid, err)
			}
		}
	}
}
