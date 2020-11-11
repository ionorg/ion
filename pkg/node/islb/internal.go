package islb

import (
	"encoding/json"
	"errors"
	"math"

	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
)

func handleRequest(rpcID string) (*nats.Subscription, error) {
	log.Infof("handleRequest: rpcID => [%s]", rpcID)
	return nrpc.Subscribe(rpcID, func(msg interface{}) (interface{}, error) {
		log.Infof("handleRequest: %T, %+v", msg, msg)

		switch v := msg.(type) {
		case *proto.ToIslbFindNodeMsg:
			return findNode(v)
		case *proto.ToIslbPeerJoinMsg:
			return peerJoin(v)
		case *proto.IslbPeerLeaveMsg:
			return peerLeave(v)
		case *proto.ToIslbStreamAddMsg:
			return streamAdd(v)
		case *proto.IslbBroadcastMsg:
			return broadcast(v)
		case *proto.ToIslbListMids:
			return listMids(v)
		default:
			return nil, errors.New("unkonw message")
		}
	})
}

// Find service nodes by name, such as sfu|avp|sip-gateway|rtmp-gateway
func findNode(data *proto.ToIslbFindNodeMsg) (interface{}, error) {
	service := data.Service
	nodes := getNodes()

	if data.RID != "" && data.UID != "" && data.MID != "" {
		mkey := proto.MediaInfo{
			DC:  dc,
			RID: data.RID,
			UID: data.UID,
			MID: data.MID,
		}.BuildKey()
		log.Infof("find mids by mkey: %s", mkey)
		for _, key := range redis.Keys(mkey + "*") {
			log.Infof("got: key => %s", key)
			minfo, err := proto.ParseMediaInfo(key)
			if err != nil {
				log.Warnf("parse media info error: %v", key)
				continue
			}
			for _, node := range nodes {
				if service == node.Service && minfo.NID == node.NID {
					log.Infof("found node by rid=% & uid=%s & mid=%s : %v", data.RID, data.UID, data.MID, node)
					return proto.FromIslbFindNodeMsg{ID: node.NID}, nil
				}
			}
		}
	}

	// MID/RID Doesn't exist in Redis
	// Find least packed node to return
	nodeID := ""
	minStreamCount := math.MaxInt32
	for _, node := range nodes {
		if service == node.Service {
			// get stream count
			nkey := proto.MediaInfo{
				DC:  dc,
				NID: node.NID,
			}.BuildKey()
			streamCount := len(redis.Keys(nkey))

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

func streamAdd(data *proto.ToIslbStreamAddMsg) (interface{}, error) {
	mkey := proto.MediaInfo{
		DC:  dc,
		RID: data.RID,
		UID: data.UID,
		MID: data.MID,
	}.BuildKey()

	field, value, err := proto.MarshalNodeField(proto.NodeInfo{
		Name: nid,
		ID:   nid,
		Type: "origin",
	})
	if err != nil {
		log.Errorf("Set: %v ", err)
	}
	err = redis.HSetTTL(mkey, field, value, redisLongKeyTTL)
	if err != nil {
		log.Errorf("Set: %v ", err)
	}

	field = "track/" + string(data.StreamID)
	// The value here actually doesn't matter, so just store the associated MID in case it's useful in the future.
	log.Infof("stores track: mkey, field, value = %s, %s, %s", mkey, field, data.MID)
	err = redis.HSetTTL(mkey, field, string(data.MID), redisLongKeyTTL)
	if err != nil {
		log.Errorf("redis.HSetTTL err = %v", err)
	}

	log.Infof("broadcast: [stream-add] => %v", data)
	err = nrpc.Publish(bid, proto.FromIslbStreamAddMsg{
		RID:    data.RID,
		UID:    data.UID,
		Stream: proto.Stream{UID: data.UID, StreamID: data.StreamID},
	})

	return nil, err
}

func listMids(data *proto.ToIslbListMids) (interface{}, error) {
	mkey := proto.MediaInfo{
		DC:  dc,
		RID: data.RID,
		UID: data.UID,
	}.BuildKey()

	mids := make([]proto.MID, 0)
	for _, key := range redis.Keys(mkey) {
		mediaInfo, err := proto.ParseMediaInfo(key)
		if err != nil {
			log.Errorf("Failed to parse media info %v", err)
			continue
		}
		mids = append(mids, mediaInfo.MID)
	}
	return proto.FromIslbListMids{MIDs: mids}, nil
}

func peerJoin(msg *proto.ToIslbPeerJoinMsg) (interface{}, error) {
	ukey := proto.UserInfo{
		DC:  dc,
		RID: msg.RID,
		UID: msg.UID,
	}.BuildKey()
	log.Infof("clientJoin: set %s => %v", ukey, string(msg.Info))

	// Tell everyone about the new peer.
	if err := nrpc.Publish(bid, proto.ToClientPeerJoinMsg{
		UID: msg.UID, RID: msg.RID, Info: msg.Info,
	}); err != nil {
		log.Errorf("broadcast peer-join error: %v", err)
		return nil, err
	}

	// Tell the new peer about everyone currently in the room.
	searchKey := proto.UserInfo{
		DC:  dc,
		RID: msg.RID,
	}.BuildKey()
	keys := redis.Keys(searchKey)

	peers := make([]proto.Peer, 0)
	streams := make([]proto.Stream, 0)
	for _, key := range keys {
		fields := redis.HGetAll(key)
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
			DC:  dc,
			RID: msg.RID,
			UID: parsedUserKey.UID,
		}.BuildKey()
		mediaKeys := redis.Keys(mkey)
		for _, mediaKey := range mediaKeys {
			mediaFields := redis.HGetAll(mediaKey)
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
	err := redis.HSetTTL(ukey, "info", string(msg.Info), redisLongKeyTTL)
	if err != nil {
		log.Errorf("redis.HSetTTL err = %v", err)
	}

	return proto.FromIslbPeerJoinMsg{
		Peers:   peers,
		Streams: streams,
	}, nil
}

func peerLeave(data *proto.IslbPeerLeaveMsg) (interface{}, error) {
	ukey := proto.UserInfo{
		DC:  dc,
		RID: data.RID,
		UID: data.UID,
	}.BuildKey()
	log.Infof("clientLeave: remove key => %s", ukey)
	err := redis.Del(ukey)
	if err != nil {
		log.Errorf("redis.Del err = %v", err)
	}

	if err := nrpc.Publish(bid, data); err != nil {
		log.Errorf("broadcast peer-leave error: %v", err)
		return nil, err
	}

	return nil, nil
}

func broadcast(data *proto.IslbBroadcastMsg) (interface{}, error) {
	if err := nrpc.Publish(bid, data); err != nil {
		log.Errorf("broadcast message error: %v", err)
	}

	return nil, nil
}
