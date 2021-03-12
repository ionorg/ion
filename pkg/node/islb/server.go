package islb

import (
	"context"
	"sync"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/db"
	ion "github.com/pion/ion/pkg/grpc/ion"
	proto "github.com/pion/ion/pkg/grpc/islb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type islbServer struct {
	proto.UnimplementedISLBServer
	Redis    *db.Redis
	nodeLock sync.Mutex
	nodes    map[string]discovery.Node
	in       *ISLB
}

func newISLBServer(in *ISLB, redis *db.Redis) *islbServer {
	return &islbServer{
		in:    in,
		Redis: redis,
		nodes: make(map[string]discovery.Node),
	}
}

// handleNodeDiscovery handle all Node from service discovery.
// This callback can observe all nodes in the ion cluster,
// TODO: Upload all node information to redis DB so that info
// can be shared when there are more than one ISLB in the later.
func (s *islbServer) handleNodeDiscovery(action string, node discovery.Node) {
	log.Debugf("handleNode: service %v, action %v => id %v, RPC %v", node.Service, action, node.ID(), node.RPC)
	s.nodeLock.Lock()
	defer s.nodeLock.Unlock()
	switch action {
	case discovery.Save:
		fallthrough
	case discovery.Update:
		s.nodes[node.ID()] = node
	case discovery.Delete:
		delete(s.nodes, node.ID())
	}
}

// FindNode find service nodes by service|nid|sid, such as sfu|avp|sip-gateway|rtmp-gateway
func (s *islbServer) FindNode(ctx context.Context, req *proto.FindNodeRequest) (*proto.FindNodeReply, error) {
	nid := req.GetNid()
	sid := req.GetSid()
	service := req.GetService()

	log.Infof("islb.FindNode: nid => %v, sid => %v, service => %v", nid, sid, service)

	nodes := []*ion.Node{}

	// find node by sid
	mkey := "*" + ".*." + sid + ".*"
	for _, key := range s.Redis.Keys(mkey) {
		fields := s.Redis.HGetAll(key)
		log.Debugf("key: %v, fields: %v", key, fields)
	}

	if len(nodes) == 0 {
		s.nodeLock.Lock()
		defer s.nodeLock.Unlock()
		// find node by nid or service
		//TODO: Add load balancing algorithm to select SFU nodes
		for _, node := range s.nodes {
			if nid == node.NID || service == node.Service {
				nodes = append(nodes, &ion.Node{
					Dc:      node.DC,
					Nid:     node.NID,
					Service: node.Service,
					Rpc: &ion.RPC{
						Protocol: string(node.RPC.Protocol),
						Addr:     node.RPC.Addr,
						//Params:   node.RPC.Params,
					},
				})
			}
		}
	}

	return &proto.FindNodeReply{
		Nodes: nodes,
	}, nil
}

//HandleSessionState handle node session status.
// key = dc/ion-sfu-1/room1/uid
// value = []
/*
func (s *islbServer) HandleSessionState(ctx context.Context, state *ion.SessionReport) (*ion.Empty, error) {
	session := state.Session
	key := session.Node.Dc + "." + session.Node.Nid + "." + session.Sid + "."
	for _, peer := range session.Peers {
		mkey := key + peer.Uid
		switch state.State {
		case ion.SessionReport_NEW:
			s.Redis.HSetTTL(mkey, "streams", "{}", redisLongKeyTTL)
		case ion.SessionReport_UPDATE:
			s.Redis.HSetTTL(mkey, "streams", "{}", redisLongKeyTTL)
		case ion.SessionReport_DELETE:
			s.Redis.HDel(mkey, "streams")
		}
	}
	return &ion.Empty{}, nil
}
*/

//PostISLBEvent Receive ISLBEvent(stream or session events) from ion-SFU, ion-AVP and ion-SIP
//the stream and session event will be save to redis db, which is used to create the
//global location of the media stream
// key = dc/ion-sfu-1/room1/uid
// value = [...stream/track info ...]
func (s *islbServer) PostISLBEvent(context.Context, *proto.ISLBEvent) (*ion.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method PostEvent not implemented")
}

//WatchISLBEvent broadcast ISLBEvent to ion-biz node.
//The stream metadata is forwarded to biz node and coupled with the peer in the client through UID
func (s *islbServer) WatchISLBEvent(*ion.Empty, proto.ISLB_WatchISLBEventServer) error {
	return status.Errorf(codes.Unimplemented, "method HandleEvent not implemented")
}

/*

func (s *islbServer) findNode(msg *proto.ToIslbFindNodeMsg) (interface{}, error) {
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

func (s *islbServer) streamAdd(msg *proto.ToIslbStreamAddMsg) (interface{}, error) {
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
*/
