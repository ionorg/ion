package avp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/google/uuid"
	iavp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v3"
)

// sfu client
type sfu struct {
	ctx    context.Context
	cancel context.CancelFunc
	client *nprotoo.Requestor
	config iavp.Config
	mu     sync.RWMutex

	addr string
	mid  proto.MID
	sid  proto.SID

	onCloseFn  func()
	transports map[string]*iavp.WebRTCTransport
}

// newSFU intializes a new SFU client
func newSFU(addr string, config iavp.Config) (*sfu, error) {
	log.Infof("Connecting to sfu: %s", addr)

	// Set up a connection to the sfu server.
	client := protoo.NewRequestor("rpc-" + addr)

	ctx, cancel := context.WithCancel(context.Background())
	return &sfu{
		ctx:    ctx,
		cancel: cancel,
		client: client,
		config: config,

		addr: addr,
		mid:  proto.MID(uuid.New().String()),

		transports: make(map[string]*iavp.WebRTCTransport),
	}, nil
}

// getTransport returns a webrtc transport for a session
func (s *sfu) getTransport(sid string) (*iavp.WebRTCTransport, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.transports[sid]

	// no transport yet, create one
	if t == nil {
		var err error
		if t, err = s.join(sid); err != nil {
			return nil, err
		}
		t.OnClose(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			delete(s.transports, sid)
			if len(s.transports) == 0 && s.onCloseFn != nil {
				s.cancel()
				s.onCloseFn()
			}
		})
		s.transports[sid] = t
	}

	return t, nil
}

// onClose handler called when sfu client is closed
func (s *sfu) onClose(f func()) {
	s.onCloseFn = f
}

func (s *sfu) join(sid string) (*iavp.WebRTCTransport, error) {
	log.Infof("Joining sfu session: %s", sid)

	t := iavp.NewWebRTCTransport(sid, s.config)

	offer, err := t.CreateOffer()
	if err != nil {
		log.Errorf("Error creating offer: %v", err)
		return nil, err
	}

	if err = t.SetLocalDescription(offer); err != nil {
		log.Errorf("Error setting local description: %v", err)
		return nil, err
	}

	rpcID := "rpc-" + nid + "-" + sid

	log.Infof("Send offer:\n %s", offer)
	resp, npErr := s.client.SyncRequest(proto.SfuClientJoin, proto.ToSfuJoinMsg{
		UID:     proto.UID(rpcID),
		MID:     s.mid,
		SID:     proto.SID(sid),
		RTCInfo: proto.RTCInfo{Jsep: offer},
	})
	if npErr != nil {
		log.Errorf("Error sending join request: %s, %v", resp, npErr)
		return nil, err
	}
	var msg proto.FromSfuJoinMsg
	if err := json.Unmarshal(resp, &msg); err != nil {
		log.Errorf("SfuClientOnJoin failed %v", err)
	}
	log.Infof("Join reply: %v", msg)
	if err := t.SetRemoteDescription(msg.Jsep); err != nil {
		log.Errorf("Error set remote description: %s", err)
		return nil, err
	}

	t.OnICECandidate(func(c *webrtc.ICECandidate) {
		log.Errorf("OnICECandidate: %v", c)
		if c == nil {
			// Gathering done
			return
		}
		s.client.AsyncRequest(proto.SfuClientTrickle, proto.SfuTrickleMsg{
			MID:       s.mid,
			Candidate: c.ToJSON(),
		})
	})

	protoo.OnRequest(rpcID, func(request nprotoo.Request, accept nprotoo.RespondFunc, reject nprotoo.RejectFunc) {
		method := request.Method
		data := request.Data
		log.Infof("handle sfu message: method => %s, data => %s", method, data)

		var result interface{}
		errResult := util.NewNpError(400, fmt.Sprintf("unknown method [%s]", method))

		switch method {
		case proto.SfuTrickleICE:
			var msg proto.SfuTrickleMsg
			if err := data.Unmarshal(&msg); err != nil {
				log.Errorf("trickle message unmarshal error: %s", err)
				errResult = util.NewNpError(415, "trickle message unmarshal error")
				break
			}
			if err = t.AddICECandidate(msg.Candidate); err != nil {
				log.Errorf("add ice candidate error: %s", err)
				errResult = util.NewNpError(415, "add ice candidate error")
				break
			}
			errResult = nil
		case proto.SfuClientOffer:
			var msg proto.SfuNegotiationMsg
			if err = data.Unmarshal(&msg); err == nil {
				log.Errorf("offer message unmarshal error: %s", err)
				errResult = util.NewNpError(415, "offer message unmarshal error")
				break
			}
			log.Infof("got remote description: %v", msg.Jsep)

			var err error
			if err = t.SetRemoteDescription(msg.Jsep); err != nil {
				log.Errorf("set remote description error: ", err)
				errResult = util.NewNpError(415, "set remote sdp error")
				break
			}

			var answer webrtc.SessionDescription
			if answer, err = t.CreateAnswer(); err != nil {
				log.Errorf("create answer error: ", err)
				errResult = util.NewNpError(415, "create answer error")
			}

			if err = t.SetLocalDescription(answer); err != nil {
				log.Errorf("set local description error: ", err)
				errResult = util.NewNpError(415, "create answer error")
			}

			log.Infof("create local description: %v", answer)

			s.client.AsyncRequest(proto.SfuClientAnswer, proto.SfuNegotiationMsg{
				MID:     msg.MID,
				RTCInfo: proto.RTCInfo{Jsep: answer},
			})
			errResult = nil
		}

		if errResult != nil {
			reject(errResult.Code, errResult.Reason)
		} else {
			accept(result)
		}
	})

	return t, nil
}
