package islb

import (
	"context"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/db"
	ion "github.com/pion/ion/proto/ion"
	islb "github.com/pion/ion/proto/islb"
	"github.com/square/go-jose/v3/json"
)

type islbServer struct {
	islb.UnimplementedISLBServer
	redis    *db.Redis
	islb     *ISLB
	conf     Config
	watchers map[string]islb.ISLB_WatchISLBEventServer
}

func newISLBServer(conf Config, in *ISLB, redis *db.Redis) *islbServer {
	return &islbServer{
		conf:     conf,
		islb:     in,
		redis:    redis,
		watchers: make(map[string]islb.ISLB_WatchISLBEventServer),
	}
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
			err := s.redis.Set(mkey, string(data), redisLongKeyTTL)
			if err != nil {
				log.Errorf("s.Redis.Set failed %v", err)
			}
		case ion.StreamEvent_REMOVE:
			err := s.redis.Del(mkey)
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
