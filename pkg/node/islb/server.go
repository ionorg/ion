package islb

import (
	"context"
	"sync"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/db"
	ion "github.com/pion/ion/pkg/grpc/ion"
	proto "github.com/pion/ion/pkg/grpc/islb"
	"github.com/square/go-jose/v3/json"
)

type islbServer struct {
	proto.UnimplementedISLBServer
	Redis    *db.Redis
	nodeLock sync.Mutex
	nodes    map[string]discovery.Node
	in       *ISLB
	conf     Config
	watchers map[string]proto.ISLB_WatchISLBEventServer
}

func newISLBServer(conf Config, in *ISLB, redis *db.Redis) *islbServer {
	return &islbServer{
		conf:     conf,
		in:       in,
		Redis:    redis,
		nodes:    make(map[string]discovery.Node),
		watchers: make(map[string]proto.ISLB_WatchISLBEventServer),
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

	// find node by sid from reids
	mkey := s.conf.Global.Dc + "/" + nid + "/" + sid
	log.Infof("islb.FindNode: mkey => %v", mkey)
	for _, key := range s.Redis.Keys(mkey) {
		value := s.Redis.Get(key)
		log.Debugf("key: %v, value: %v", key, value)
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

//PostISLBEvent Receive ISLBEvent(stream or session events) from ion-SFU, ion-AVP and ion-SIP
//the stream and session event will be save to redis db, which is used to create the
//global location of the media stream
// key = dc/ion-sfu-1/room1/uid
// value = [...stream/track info ...]
func (s *islbServer) PostISLBEvent(ctx context.Context, event *proto.ISLBEvent) (*ion.Empty, error) {
	log.Infof("ISLBServer.PostISLBEvent")
	switch payload := event.Payload.(type) {
	case *proto.ISLBEvent_Stream:
		stream := payload.Stream
		state := stream.State
		mkey := s.conf.Global.Dc + "/" + stream.Nid + "/" + stream.Sid + "/" + stream.Uid
		data, err := json.Marshal(stream.Streams)
		if err != nil {
			log.Errorf("json.Marshal err => %v", err)
		}

		jstr, err := json.MarshalIndent(stream.Streams, "", "  ")
		if err != nil {
			log.Errorf("json.MarshalIndent failed %v", err)
		}
		log.Infof("ISLBEvent:\nmkey=> %v\nstate = %v\nstreams => %v", mkey, state.String(), string(jstr))

		switch state {
		case ion.StreamEvent_ADD:
			s.Redis.Set(mkey, string(data), redisLongKeyTTL)
		case ion.StreamEvent_REMOVE:
			s.Redis.Del(mkey)
		}

		for _, wstream := range s.watchers {
			wstream.Send(event)
		}

	case *proto.ISLBEvent_Session:
		//session := payload.Session
		//log.Infof("ISLBEvent_Session event %v", session.String())
	}
	return &ion.Empty{}, nil
}

//WatchISLBEvent broadcast ISLBEvent to ion-biz node.
//The stream metadata is forwarded to biz node and coupled with the peer in the client through UID
func (s *islbServer) WatchISLBEvent(stream proto.ISLB_WatchISLBEventServer) error {
	var sid string
	defer func() {
		delete(s.watchers, sid)
	}()
	for {
		req, err := stream.Recv()
		if err != nil {
			log.Errorf("ISLBServer.WatchISLBEvent server stream.Recv() err: %v", err)
			return err
		}
		log.Infof("ISLBServer.WatchISLBEvent req => %v", req)
		sid = req.Sid
		if _, found := s.watchers[sid]; !found {
			s.watchers[sid] = stream
		}
	}
}
