package islb

import (
	"encoding/json"
	"errors"
	"math"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/proto"
)

type server struct {
	dc       string
	nid      string
	bid      string
	nrpc     *proto.NatsRPC
	sub      *nats.Subscription
	redis    *db.Redis
	getNodes func() map[string]discovery.Node
}

func newServer(dc, nid string, nrpc *proto.NatsRPC, getNodes func() map[string]discovery.Node) *server {
	return &server{
		dc:       dc,
		nid:      nid,
		bid:      nid + "-event",
		nrpc:     nrpc,
		getNodes: getNodes,
	}
}

func (s *server) start(conf db.Config) error {
	var err error

	s.redis = db.NewRedis(conf)
	if s.redis == nil {
		return errors.New("new redis error")
	}

	if s.sub, err = s.nrpc.QueueSubscribe(proto.ISLB(s.dc), s.dc, s.handle); err != nil {
		return err
	}
	return nil
}

func (s *server) close() {
	if s.sub != nil {
		if err := s.sub.Unsubscribe(); err != nil {
			log.Errorf("unsubscribe %s error: %v", s.sub.Subject, err)
		}
	}
	if s.redis != nil {
		s.redis.Close()
	}
}

func (s *server) handle(msg interface{}) (interface{}, error) {
	log.Infof("handleRequest: %T, %+v", msg, msg)

	switch v := msg.(type) {
	case *proto.ToIslbFindNodeMsg:
		return s.findNode(v)
	case *proto.ToIslbPeerJoinMsg:
		return s.peerJoin(v)
	case *proto.IslbPeerLeaveMsg:
		return s.peerLeave(v)
	case *proto.ToIslbStreamAddMsg:
		return s.streamAdd(v)
	case *proto.IslbBroadcastMsg:
		return s.broadcast(v)
	default:
		return nil, errors.New("unkonw message")
	}
}

// Find service nodes by name, such as sfu|avp|sip-gateway|rtmp-gateway
func (s *server) findNode(msg *proto.ToIslbFindNodeMsg) (interface{}, error) {
	service := msg.Service
	nodes := s.getNodes()

	if msg.SID != "" && msg.UID != "" && msg.MID != "" {
		mkey := proto.MediaInfo{
			DC:  s.dc,
			SID: msg.SID,
			UID: msg.UID,
			MID: msg.MID,
		}.BuildKey()
		log.Infof("find mids by mkey: %s", mkey)
		for _, key := range s.redis.Keys(mkey + "*") {
			log.Infof("got: key => %s", key)
			minfo, err := proto.ParseMediaInfo(key)
			if err != nil {
				log.Warnf("parse media info error: %v", key)
				continue
			}
			for _, node := range nodes {
				if service == node.Service && minfo.NID == node.NID {
					log.Infof("found node by sid=%s & uid=%s & mid=%s : %v", msg.SID, msg.UID, msg.MID, node)
					return proto.FromIslbFindNodeMsg{ID: node.NID}, nil
				}
			}
		}
	}

	// MID/SID Doesn't exist in Redis
	// Find least packed node to return
	nodeID := ""
	minStreamCount := math.MaxInt32
	for _, node := range nodes {
		if service == node.Service {
			// get stream count
			nkey := proto.MediaInfo{
				DC:  s.dc,
				NID: node.NID,
			}.BuildKey()
			streamCount := len(s.redis.Keys(nkey))

			log.Infof("looking up node stream count: [%s] = %v", nkey, streamCount)
			if streamCount <= minStreamCount {
				nodeID = node.NID
				minStreamCount = streamCount
			}
		}
	}
	log.Infof("selecting node: [%s] = %v", nodeID, minStreamCount)
	if node, ok := nodes[nodeID]; ok {
		log.Infof("found best node: %v", node)
		return proto.FromIslbFindNodeMsg{ID: node.NID}, nil
	}

	// TODO: Add a load balancing algorithm.
	for _, node := range nodes {
		if service == node.Service {
			log.Infof("found node: %v", node)
			return proto.FromIslbFindNodeMsg{ID: node.NID}, nil
		}
	}

	return nil, errors.New("service node not found")
}

func (s *server) streamAdd(msg *proto.ToIslbStreamAddMsg) (interface{}, error) {
	mkey := proto.MediaInfo{
		DC:  s.dc,
		SID: msg.SID,
		UID: msg.UID,
		MID: msg.MID,
	}.BuildKey()

	field, value, err := proto.MarshalNodeField(proto.NodeInfo{
		Name: s.nid,
		ID:   s.nid,
		Type: "origin",
	})
	if err != nil {
		log.Errorf("Set: %v ", err)
	}
	err = s.redis.HSetTTL(mkey, field, value, redisLongKeyTTL)
	if err != nil {
		log.Errorf("Set: %v ", err)
	}

	field = "track/" + string(msg.StreamID)
	// The value here actually doesn't matter, so just store the associated MID in case it's useful in the future.
	log.Infof("stores track: mkey, field, value = %s, %s, %s", mkey, field, msg.MID)
	err = s.redis.HSetTTL(mkey, field, string(msg.MID), redisLongKeyTTL)
	if err != nil {
		log.Errorf("redis.HSetTTL err = %v", err)
	}

	log.Infof("broadcast: [stream-add] => %v", msg)
	err = s.nrpc.Publish(s.bid, proto.FromIslbStreamAddMsg{
		SID:    msg.SID,
		UID:    msg.UID,
		Stream: proto.Stream{UID: msg.UID, StreamID: msg.StreamID},
	})

	return nil, err
}

func (s *server) peerJoin(msg *proto.ToIslbPeerJoinMsg) (interface{}, error) {
	ukey := proto.UserInfo{
		DC:  s.dc,
		SID: msg.SID,
		UID: msg.UID,
	}.BuildKey()
	log.Infof("clientJoin: set %s => %v", ukey, string(msg.Info))

	// Tell everyone about the new peer.
	if err := s.nrpc.Publish(s.bid, proto.ToClientPeerJoinMsg{
		UID: msg.UID, SID: msg.SID, Info: msg.Info,
	}); err != nil {
		log.Errorf("broadcast peer-join error: %v", err)
		return nil, err
	}

	// Tell the new peer about everyone currently in the room.
	searchKey := proto.UserInfo{
		DC:  s.dc,
		SID: msg.SID,
	}.BuildKey()
	keys := s.redis.Keys(searchKey)

	peers := make([]proto.Peer, 0)
	streams := make([]proto.Stream, 0)
	for _, key := range keys {
		fields := s.redis.HGetAll(key)
		parsedUserKey, err := proto.ParseUserInfo(key)
		if err != nil {
			log.Errorf("redis.HGetAll err = %v", err)
			continue
		}
		if info, ok := fields["info"]; ok {
			peers = append(peers, proto.Peer{
				UID:  parsedUserKey.UID,
				Info: json.RawMessage(info),
			})
		} else {
			log.Warnf("No info found for %v", key)
		}

		mkey := proto.MediaInfo{
			DC:  s.dc,
			SID: msg.SID,
			UID: parsedUserKey.UID,
		}.BuildKey()
		mediaKeys := s.redis.Keys(mkey)
		for _, mediaKey := range mediaKeys {
			mediaFields := s.redis.HGetAll(mediaKey)
			for mediaField := range mediaFields {
				log.Warnf("Received media field %s for key %s", mediaField, mediaKey)
				if len(mediaField) > 6 && mediaField[:6] == "track/" {
					streams = append(streams, proto.Stream{
						UID:      parsedUserKey.UID,
						StreamID: proto.StreamID(mediaField[6:]),
					})
				}
			}
		}
	}

	// Write the user info to redis.
	err := s.redis.HSetTTL(ukey, "info", string(msg.Info), redisLongKeyTTL)
	if err != nil {
		log.Errorf("redis.HSetTTL err = %v", err)
	}

	return proto.FromIslbPeerJoinMsg{
		Peers:   peers,
		Streams: streams,
	}, nil
}

func (s *server) peerLeave(msg *proto.IslbPeerLeaveMsg) (interface{}, error) {
	ukey := proto.UserInfo{
		DC:  s.dc,
		SID: msg.SID,
		UID: msg.UID,
	}.BuildKey()
	log.Infof("clientLeave: remove key => %s", ukey)
	err := s.redis.Del(ukey)
	if err != nil {
		log.Errorf("redis.Del err = %v", err)
	}

	if err := s.nrpc.Publish(s.bid, msg); err != nil {
		log.Errorf("broadcast peer-leave error: %v", err)
		return nil, err
	}

	return nil, nil
}

func (s *server) broadcast(msg *proto.IslbBroadcastMsg) (interface{}, error) {
	if err := s.nrpc.Publish(s.bid, msg); err != nil {
		log.Errorf("broadcast message error: %v", err)
	}

	return nil, nil
}
