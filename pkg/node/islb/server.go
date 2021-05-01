package islb

import (
	"context"
	"sync"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/cloudwebrtc/nats-discovery/pkg/registry"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/db"
	ion "github.com/pion/ion/pkg/grpc/ion"
	islb "github.com/pion/ion/pkg/grpc/islb"
	"github.com/pion/ion/pkg/proto"
	"github.com/square/go-jose/v3/json"
)

type islbServer struct {
	islb.UnimplementedISLBServer
	Redis    *db.Redis
	registry *registry.Registry
	nodeLock sync.Mutex
	nodes    map[string]discovery.Node
	in       *ISLB
	conf     Config
	watchers map[string]islb.ISLB_WatchISLBEventServer
}

func newISLBServer(conf Config, in *ISLB, redis *db.Redis, registry *registry.Registry) *islbServer {
	return &islbServer{
		conf:     conf,
		in:       in,
		Redis:    redis,
		registry: registry,
		nodes:    make(map[string]discovery.Node),
		watchers: make(map[string]islb.ISLB_WatchISLBEventServer),
	}
}

// handleNodeDiscovery handle all Node from service discovery.
// This callback can observe all nodes in the ion cluster,
// TODO: Upload all node information to redis DB so that info
// can be shared when there are more than one ISLB in the later.
func (s *islbServer) handleNodeDiscovery(action discovery.Action, node discovery.Node) (bool, error) {
	//Add authentication here
	log.Debugf("handleNode: service %v, action %v => id %v, RPC %v", node.Service, action, node.ID(), node.RPC)

	//TODO: Put node info into the redis.
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

	return true, nil
}

func (s *islbServer) handleGetNodes(service string, params map[string]interface{}) ([]discovery.Node, error) {
	//Add load balancing here.
	log.Infof("Get node by %v, params %v", service, params)

	if service == proto.ServiceSFU {

		nid := "*"
		sid := ""

		if val, ok := params["nid"]; ok {
			nid = val.(string)
		}

		if val, ok := params["sid"]; ok {
			sid = val.(string)
		}

		// find node by nid/sid from reids
		mkey := s.conf.Global.Dc + "/" + nid + "/" + sid
		log.Infof("islb.FindNode: mkey => %v", mkey)
		for _, key := range s.Redis.Keys(mkey) {
			value := s.Redis.Get(key)
			log.Debugf("key: %v, value: %v", key, value)
		}
	}

	return s.registry.GetNodes(service)
}

//PostISLBEvent Receive ISLBEvent(stream or session events) from ion-SFU, ion-AVP and ion-SIP
//the stream and session event will be save to redis db, which is used to create the
//global location of the media stream
// key = dc/ion-sfu-1/room1/uid
// value = [...stream/track info ...]
func (s *islbServer) PostISLBEvent(ctx context.Context, event *islb.ISLBEvent) (*ion.Empty, error) {
	log.Infof("ISLBServer.PostISLBEvent")
	switch payload := event.Payload.(type) {
	case *islb.ISLBEvent_Stream:
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
			err := s.Redis.Set(mkey, string(data), redisLongKeyTTL)
			if err != nil {
				log.Errorf("s.Redis.Set failed %v", err)
			}
		case ion.StreamEvent_REMOVE:
			err := s.Redis.Del(mkey)
			if err != nil {
				log.Errorf("s.Redis.Del failed %v", err)
			}
		}

		for _, wstream := range s.watchers {
			err := wstream.Send(event)
			if err != nil {
				log.Errorf("wstream.Send(event): failed %v", err)
			}
		}

	case *islb.ISLBEvent_Session:
		//session := payload.Session
		//log.Infof("ISLBEvent_Session event %v", session.String())
	}
	return &ion.Empty{}, nil
}

//WatchISLBEvent broadcast ISLBEvent to ion-biz node.
//The stream metadata is forwarded to biz node and coupled with the peer in the client through UID
func (s *islbServer) WatchISLBEvent(stream islb.ISLB_WatchISLBEventServer) error {
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
