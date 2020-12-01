package biz

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/notedit/sdp"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v3"
	"github.com/sourcegraph/jsonrpc2"
	websocketjsonrpc2 "github.com/sourcegraph/jsonrpc2/websocket"
)

// peer represents a peer for client
type peer struct {
	uid        proto.UID
	mid        proto.MID
	rid        proto.RID
	info       []byte
	conn       *jsonrpc2.Conn
	ctx        context.Context
	leaveOnce  sync.Once
	closed     util.AtomicBool
	onCloseFun func()
	auth       func(proto.Authenticatable) error
}

// newPeer create peer instance for client
func newPeer(ctx context.Context, c *websocket.Conn, auth func(proto.Authenticatable) error) *peer {
	id := uuid.New().String()
	p := &peer{
		ctx:  ctx,
		uid:  proto.UID(id), // TODO: may be improve
		mid:  proto.MID(id),
		auth: auth,
	}
	p.conn = jsonrpc2.NewConn(ctx, websocketjsonrpc2.NewObjectStream(c), p)
	return p
}

// Handle incoming RPC call events, implement jsonrpc2.Handler
func (p *peer) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	replyError := func(err error) {
		_ = conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    500,
			Message: fmt.Sprintf("%s", err),
		})
	}

	switch req.Method {
	case proto.ClientJoin:
		var msg proto.FromClientJoinMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing offer: %v", err)
			replyError(err)
			break
		}
		answer, err := p.join(&msg)
		if err != nil {
			replyError(err)
			break
		}
		_ = conn.Reply(ctx, req.ID, answer)

	case proto.ClientOffer:
		var msg proto.ClientOfferMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing trickle: %v", err)
			replyError(err)
		}
		answer, err := p.offer(&msg)
		if err != nil {
			replyError(err)
			break
		}
		_ = conn.Reply(ctx, req.ID, answer)

	case proto.ClientAnswer:
		var msg proto.ClientAnswerMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing trickle: %v", err)
			replyError(err)
			break
		}
		if err := p.answer(&msg); err != nil {
			replyError(err)
		}

	case proto.ClientTrickle:
		var msg proto.ClientTrickleMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing trickle: %v", err)
			replyError(err)
			break
		}
		if err := p.trickle(&msg); err != nil {
			replyError(err)
		}

	case proto.ClientLeave:
		var msg proto.FromClientLeaveMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing leave: %v", err)
			replyError(err)
			break
		}
		if err := p.leave(&msg); err != nil {
			replyError(err)
		}

	case "broadcast":
		var msg proto.FromClientBroadcastMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing trickle: %v", err)
			replyError(err)
			break
		}
		if err := p.broadcast(&msg); err != nil {
			replyError(err)
		}

	default:
		replyError(errors.New("unknown message"))
	}
}

// handleSFURequest handle sfu request
func (p *peer) handleSFURequest(islb, sfu string) {
	sub, err := nrpc.Subscribe(string(p.mid), func(msg interface{}) (interface{}, error) {
		log.Infof("peer(%s) handle sfu message: %T, %+v", p.uid, msg, msg)
		switch v := msg.(type) {
		case *proto.SfuOfferMsg:
			log.Infof("peer(%s) got remote description: %v", p.uid, v.Desc)

			// join to islb
			resp, err := nrpc.Request(islb, proto.ToIslbPeerJoinMsg{
				UID: p.uid, RID: p.rid, MID: p.mid, Info: p.info,
			})
			if err != nil {
				log.Errorf("IslbClientOnJoin failed %v", err)
			}
			fromIslbPeerJoinMsg := resp.(*proto.FromIslbPeerJoinMsg)
			if err := p.notify(proto.ClientPeers, proto.ToClientPeersMsg{
				Peers:   fromIslbPeerJoinMsg.Peers,
				Streams: fromIslbPeerJoinMsg.Streams,
			}); err != nil {
				log.Errorf("error sending offer %s", err)
			}

			// join to avp
			var avp string
			if len(avpElements) > 0 {
				if avp, err = getNode("avp", islb, p.uid, p.rid, p.mid); err != nil {
					log.Errorf("get avp-node error: %v", err)
				}
			}
			if avp != "" {
				sdpInfo, err := sdp.Parse(v.Desc.SDP)
				if err != nil {
					log.Errorf("parse sdp error: %v", err)
				}
				for _, eid := range avpElements {
					for _, stream := range sdpInfo.GetStreams() {
						tracks := stream.GetTracks()
						for _, track := range tracks {
							err = nrpc.Publish(avp, proto.ToAvpProcessMsg{
								Addr:   sfu,
								PID:    stream.GetID(),
								RID:    string(p.rid),
								TID:    track.GetID(),
								EID:    eid,
								Config: []byte{},
							})
							if err != nil {
								log.Errorf("avp process failed %v", err)
							}
						}
					}
				}
			}

			if err := p.notify(proto.ClientOffer, v.Desc); err != nil {
				log.Errorf("error sending offer %s", err)
			}
		case *proto.SfuTrickleMsg:
			log.Infof("peer(%s) got a remote candidate: %v", p.uid, v.Candidate)
			if err := p.notify(proto.ClientTrickle, proto.ClientTrickleMsg{
				Candidate: proto.CandidateForJSON(v.Candidate),
				Target:    v.Target,
			}); err != nil {
				log.Errorf("error sending ice candidate %s", err)
			}
		case *proto.SfuICEConnectionStateMsg:
			log.Infof("peer(%s) got ice connection state: %v", p.uid, v.State)
			switch v.State {
			case webrtc.ICEConnectionStateFailed:
				fallthrough
			case webrtc.ICEConnectionStateClosed:
				p.leaveOnce.Do(func() {
					if err := p.leave(&proto.FromClientLeaveMsg{RID: p.rid}); err != nil {
						log.Infof("peer(%s) leave % error: %v", p.rid, err)
					}
				})
			}
		default:
			return nil, errors.New("unkonw message")
		}
		return nil, nil
	})
	if err != nil {
		log.Errorf("subscribe sfu failed: %v", err)
	}

	p.setCloseFun(func() {
		if err := sub.Unsubscribe(); err != nil {
			log.Errorf("unsubscribe %s error: %v", p.mid, err)
		}
	})
}

// join client join the room
func (p *peer) join(msg *proto.FromClientJoinMsg) (interface{}, error) {
	log.Infof("peer join: uid=%s, msg=%v", p.uid, msg)

	p.rid = msg.RID
	p.info = msg.Info

	// validate
	if p.rid == "" {
		return nil, errors.New("room not found")
	}

	// join room
	addPeer(p.rid, p)

	// get islb and sfu node
	islb := getIslb()
	if islb == "" {
		return nil, errors.New("islb-node not found")
	}
	sfu, err := getNode("sfu", islb, p.uid, p.rid, p.mid)
	if err != nil {
		log.Errorf("getting sfu-node: %v", err)
		return nil, errors.New("sfu-node not found")
	}

	// handle sfu message
	p.handleSFURequest(islb, sfu)

	// join to sfu
	resp, err := nrpc.Request(sfu, proto.ToSfuJoinMsg{
		MID:   p.mid,
		RID:   p.rid,
		Offer: msg.Offer,
	})
	if err != nil {
		return nil, err
	}
	fromSfuJoinMsg := resp.(*proto.FromSfuJoinMsg)

	return fromSfuJoinMsg.Answer, nil
}

// offer client send offer to biz
func (p *peer) offer(msg *proto.ClientOfferMsg) (interface{}, error) {
	log.Infof("peer offer: uid=%s, msg=%v", p.uid, msg)

	islb := getIslb()
	if islb == "" {
		return nil, errors.New("islb-node not found")
	}

	// associate the stream in the SDP with the UID/RID/MID.
	sdpInfo, err := sdp.Parse(msg.Desc.SDP)
	if err != nil {
		log.Errorf("parse sdp error: %v", err)
	}
	for key := range sdpInfo.GetStreams() {
		if err := nrpc.Publish(islb, proto.ToIslbStreamAddMsg{
			UID: p.uid, RID: p.rid, MID: p.mid, StreamID: proto.StreamID(key),
		}); err != nil {
			log.Errorf("send stream-add to %s error: %v", islb, err)
		}
	}

	sfu, err := getNode("sfu", islb, p.uid, p.rid, p.mid)
	if err != nil {
		log.Warnf("sfu-node not found, %s", err.Error())
		return nil, err
	}

	resp, err := nrpc.Request(sfu, proto.SfuOfferMsg{
		MID:  p.mid,
		Desc: msg.Desc,
	})
	if err != nil {
		log.Errorf("offer %s failed %v", sfu, err.Error())
		return nil, err
	}

	return resp.(*proto.SfuAnswerMsg).Desc, nil
}

// answer received answer of client
func (p *peer) answer(msg *proto.ClientAnswerMsg) error {
	log.Infof("peer answer:  uid=%s, msg=%v", p.uid, msg)

	sfu, err := getNode("sfu", "", p.uid, p.rid, p.mid)
	if err != nil {
		log.Warnf("sfu-node not found, %s", err.Error())
		return err
	}

	if _, err := nrpc.Request(sfu, proto.SfuAnswerMsg{
		MID:  p.mid,
		Desc: msg.Desc,
	}); err != nil {
		log.Errorf("answer %s error: %v", sfu, err.Error())
		return err
	}

	return nil
}

// trickle received candidate of client
func (p *peer) trickle(msg *proto.ClientTrickleMsg) error {
	log.Infof("peer trickle: uid=%s, msg=%v", p.uid, msg)

	sfu, err := getNode("sfu", "", p.uid, p.rid, p.mid)
	if err != nil {
		log.Warnf("sfu-node not found, %s", err.Error())
		return err
	}

	_, err = nrpc.Request(sfu, proto.SfuTrickleMsg{
		MID:       p.mid,
		Candidate: msg.Candidate,
		Target:    msg.Target,
	})
	if err != nil {
		log.Errorf("trickle %s error: %s", sfu, err.Error())
		return err
	}

	return nil
}

// leave client leave the room
func (p *peer) leave(msg *proto.FromClientLeaveMsg) error {
	log.Infof("peer leave: uid=%s, msg=%v", p.uid, msg)

	// leave room
	room := getRoom(msg.RID)
	if room == nil {
		log.Warnf("room not exits, rid=%s", msg.RID)
		return errors.New("room not found")
	}
	room.delPeer(p.uid)

	islb := getIslb()
	if islb == "" {
		log.Errorf("islb-node not found")
		return errors.New("islb-node not found")
	}

	if _, err := nrpc.Request(islb, proto.IslbPeerLeaveMsg{
		RoomInfo: proto.RoomInfo{UID: p.uid, RID: msg.RID},
	}); err != nil {
		log.Errorf("leave %s error: %v", islb, err.Error())
	}

	sfu, err := getNode("sfu", islb, p.uid, msg.RID, p.mid)
	if err != nil {
		log.Errorf("sfu-node not found: %s", err)
	}
	if _, err := nrpc.Request(sfu, proto.ToSfuLeaveMsg{
		MID: p.mid,
	}); err != nil {
		log.Errorf("leave %s error: %v", sfu, err.Error())
	}

	return nil
}

// broadcast peer send message to peers of room
func (p *peer) broadcast(msg *proto.FromClientBroadcastMsg) error {
	log.Infof("peer broadcast: uid=%s, msg=%v", p.uid, msg)

	// validate
	if msg.RID == "" {
		return errors.New("room not found")
	}

	islb := getIslb()
	if islb == "" {
		return errors.New("islb-node not found")
	}

	// TODO: nrpc.Publish(roomID, ...
	err := nrpc.Publish(islb, proto.IslbBroadcastMsg{
		RoomInfo: proto.RoomInfo{UID: p.uid, RID: msg.RID},
		Info:     msg.Info,
	})
	if err != nil {
		log.Errorf("broadcast error: %s", err.Error())
		return err
	}

	return nil
}

// unmarshal message
func (p *peer) unmarshal(data json.RawMessage, result interface{}) error {
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	if p.auth != nil {
		if msg, ok := result.(proto.Authenticatable); ok {
			return p.auth(msg)
		}
	}
	return nil
}

// // request send msg to message and waits for the response
// func (p *peer) request(method string, params, result interface{}) error {
// 	return p.conn.Call(p.ctx, method, params, result)
// }

// notify send a message to the peer
func (p *peer) notify(method string, params interface{}) error {
	return p.conn.Notify(p.ctx, method, params)
}

// disconnectNotify returns a channel that is closed when the
// underlying connection is disconnected.
func (p *peer) disconnectNotify() <-chan struct{} {
	return p.conn.DisconnectNotify()
}

// close peer
func (p *peer) close() {
	if p.closed.Get() {
		return
	}
	p.closed.Set(true)

	// leave all rooms
	if err := p.leave(&proto.FromClientLeaveMsg{RID: p.rid}); err != nil {
		log.Infof("peer(%s) leave % error: %v", p.rid, err)
	}

	if p.onCloseFun != nil {
		p.onCloseFun()
	}
}

// setCloseFun sets a handler that is called when the peer close
func (p *peer) setCloseFun(f func()) {
	p.onCloseFun = f
}
