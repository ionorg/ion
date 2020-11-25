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
	"github.com/sourcegraph/jsonrpc2"
	websocketjsonrpc2 "github.com/sourcegraph/jsonrpc2/websocket"
)

// peer represents a peer
type peer struct {
	id         proto.UID
	conn       *jsonrpc2.Conn
	ctx        context.Context
	closed     util.AtomicBool
	onCloseFun func()
	rooms      map[proto.RID]proto.MID
	roomLook   sync.Mutex
	auth       func(proto.Authenticatable) error
}

// newPeer create peer instance for client
func newPeer(ctx context.Context, c *websocket.Conn, id proto.UID, auth func(proto.Authenticatable) error) *peer {
	p := &peer{
		ctx:   ctx,
		id:    id,
		rooms: make(map[proto.RID]proto.MID),
		auth:  auth,
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
	case "join":
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

	case "offer":
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

	case "answer":
		var msg proto.ClientAnswerMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing trickle: %v", err)
			replyError(err)
			break
		}
		if err := p.answer(&msg); err != nil {
			replyError(err)
		}

	case "trickle":
		var msg proto.ClientTrickleMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing trickle: %v", err)
			replyError(err)
			break
		}
		if err := p.trickle(&msg); err != nil {
			replyError(err)
		}

	case "leave":
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

// join client join the room
func (p *peer) join(msg *proto.FromClientJoinMsg) (interface{}, error) {
	log.Infof("peer join: uid=%s, msg=%v", p.id, msg)

	rid := proto.RID(msg.RID)
	uid := p.id

	// validate
	if rid == "" {
		return nil, errors.New("room not found")
	}
	sdpInfo, err := sdp.Parse(msg.Jsep.SDP)
	if err != nil {
		return nil, errors.New("sdp not found")
	}

	// join room
	p.roomLook.Lock()
	mid, joined := p.rooms[rid]
	if !joined {
		mid = proto.MID(uuid.New().String())
		p.rooms[rid] = mid
	}
	p.roomLook.Unlock()
	if joined {
		return nil, errors.New("peer already exists")
	}
	addPeer(rid, p)

	// get islb and sfu node
	islb := getIslb()
	if islb == "" {
		return nil, errors.New("islb-node not found")
	}
	sfu, err := getNode("sfu", islb, uid, rid, mid)
	if err != nil {
		log.Errorf("getting sfu-node: %v", err)
		return nil, errors.New("sfu-node not found")
	}

	// join to islb
	resp, err := nrpc.Request(islb, proto.ToIslbPeerJoinMsg{
		UID: uid, RID: rid, MID: mid, Info: msg.Info,
	})
	if err != nil {
		log.Errorf("IslbClientOnJoin failed %v", err)
	}
	fromIslbPeerJoinMsg, ok := resp.(*proto.FromIslbPeerJoinMsg)
	if !ok {
		log.Errorf("IslbClientOnJoin failed %v", fromIslbPeerJoinMsg)
	}

	// handle sfu message
	rpcID := string(uid)
	sub, err := nrpc.Subscribe(rpcID, func(msg interface{}) (interface{}, error) {
		log.Infof("peer(%s) handle sfu message: %+v", uid, msg)
		switch v := msg.(type) {
		case *proto.SfuOfferMsg:
			log.Infof("peer(%s) got remote description: %s", uid, v.Jsep)
			if err := p.notify(proto.ClientOffer, proto.ClientOfferMsg{
				RID:     rid,
				MID:     v.MID,
				RTCInfo: v.RTCInfo,
			}); err != nil {
				log.Errorf("error sending offer %s", err)
			}
		case *proto.SfuTrickleMsg:
			log.Infof("peer(%s) got a remote candidate: %s", uid, v.Candidate)
			if err := p.notify(proto.ClientTrickleICE, proto.ClientTrickleMsg{
				RID:       rid,
				MID:       v.MID,
				Candidate: proto.CandidateForJSON(v.Candidate),
			}); err != nil {
				log.Errorf("error sending ice candidate %s", err)
			}
		default:
			return nil, errors.New("unkonw message")
		}
		return nil, nil
	})
	if err != nil {
		log.Errorf("subscribe sfu failed: %v", err)
		return nil, errors.New("subscribe sfu failed")
	}
	p.setCloseFun(func() {
		sub.Unsubscribe()
	})

	// join to sfu
	resp, err = nrpc.Request(sfu, proto.ToSfuJoinMsg{
		RPCID:   rpcID,
		MID:     mid,
		RID:     rid,
		RTCInfo: msg.RTCInfo,
	})
	if err != nil {
		return nil, err
	}
	fromSfuJoinMsg, ok := resp.(*proto.FromSfuJoinMsg)
	if !ok {
		return nil, errors.New("join reply msg parses failed")
	}

	// associate the stream in the SDP with the UID/RID/MID.
	for key := range sdpInfo.GetStreams() {
		nrpc.Publish(islb, proto.ToIslbStreamAddMsg{
			UID: uid, RID: rid, MID: mid, StreamID: proto.StreamID(key),
		})
	}

	// join to avp
	var avp string
	if len(avpElements) > 0 {
		if avp, err = getNode("avp", islb, uid, rid, mid); err != nil {
			log.Errorf("get avp-node error: %v", err)
		}
	}
	if avp != "" {
		for _, eid := range avpElements {
			for _, stream := range sdpInfo.GetStreams() {
				tracks := stream.GetTracks()
				for _, track := range tracks {
					err = nrpc.Publish(avp, proto.ToAvpProcessMsg{
						Addr:   sfu,
						PID:    stream.GetID(),
						RID:    string(rid),
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

	return proto.ToClientJoinMsg{
		Peers:   fromIslbPeerJoinMsg.Peers,
		Streams: fromIslbPeerJoinMsg.Streams,
		MID:     mid,
		RTCInfo: fromSfuJoinMsg.RTCInfo,
	}, nil
}

// offer client send offer to biz
func (p *peer) offer(msg *proto.ClientOfferMsg) (interface{}, error) {
	log.Infof("peer offer: uid=%s, msg=%v", p.id, msg)

	sfu, err := getNode("sfu", "", p.id, msg.RID, msg.MID)
	if err != nil {
		log.Warnf("sfu-node not found, %s", err.Error())
		return nil, err
	}

	resp, err := nrpc.Request(sfu, proto.SfuOfferMsg{
		MID:     msg.MID,
		RTCInfo: proto.RTCInfo{Jsep: msg.Jsep},
	})
	if err != nil {
		log.Errorf("SfuClientOnOffer failed %v", err.Error())
		return nil, err
	}
	answer, ok := resp.(*proto.SfuAnswerMsg)
	if !ok {
		log.Errorf("parse answer failed")
		return nil, errors.New("parse answer failed")
	}

	return proto.ClientAnswerMsg{
		RID:     msg.RID,
		MID:     msg.MID,
		RTCInfo: answer.RTCInfo,
	}, nil
}

// answer received answer of client
func (p *peer) answer(msg *proto.ClientAnswerMsg) error {
	log.Infof("peer answer:  uid=%s, msg=%v", p.id, msg)

	sfu, err := getNode("sfu", "", p.id, msg.RID, msg.MID)
	if err != nil {
		log.Warnf("sfu-node not found, %s", err.Error())
		return err
	}

	if _, err := nrpc.Request(sfu, proto.SfuAnswerMsg{
		MID:     msg.MID,
		RTCInfo: msg.RTCInfo,
	}); err != nil {
		log.Errorf("SfuAnswerMsg failed %v", err.Error())
		return err
	}

	return nil
}

// trickle received candidate of client
func (p *peer) trickle(msg *proto.ClientTrickleMsg) error {
	log.Infof("peer trickle: uid=%s, msg=%v", p.id, msg)

	if msg.RID == "" {
		return errors.New("room not found")
	}

	sfu, err := getNode("sfu", "", p.id, msg.RID, msg.MID)
	if err != nil {
		log.Warnf("sfu-node not found, %s", err.Error())
		return err
	}

	_, err = nrpc.Request(sfu, proto.SfuTrickleMsg{
		MID:       msg.MID,
		Candidate: msg.Candidate,
	})
	if err != nil {
		log.Errorf("send trickle to sfu error: %s", err.Error())
		return err
	}

	return nil
}

// leave client leave the room
func (p *peer) leave(msg *proto.FromClientLeaveMsg) error {
	log.Infof("peer leave: msg=%v", p.id, msg)

	// leave room
	p.roomLook.Lock()
	delete(p.rooms, msg.RID)
	p.roomLook.Unlock()
	room := getRoom(msg.RID)
	if room == nil {
		log.Warnf("room not exits, rid=", msg.RID)
		return errors.New("room not found")
	}
	room.DelPeer(msg.UID)

	islb := getIslb()
	if islb == "" {
		log.Errorf("islb-node not found")
		return errors.New("islb-node not found")
	}

	if _, err := nrpc.Request(islb, proto.IslbPeerLeaveMsg{
		RoomInfo: proto.RoomInfo{UID: msg.UID, RID: msg.RID},
	}); err != nil {
		log.Errorf("IslbPeerLeave error: %v", err.Error())
	}

	var mids []proto.MID
	if msg.MID == "" {
		if resp, err := nrpc.Request(islb, proto.ToIslbListMids{
			UID: msg.UID,
			RID: msg.RID,
		}); err == nil {
			if v, ok := resp.(*proto.FromIslbListMids); ok {
				mids = v.MIDs
			} else {
				log.Errorf("json.Unmarshal error: %v", err)
			}
		} else {
			log.Errorf("IslbListMids error: %v", err)
		}
	} else {
		mids = append(mids, msg.MID)
	}

	for _, mid := range mids {
		sfu, err := getNode("sfu", islb, msg.UID, msg.RID, mid)
		if err != nil {
			log.Errorf("sfu-node not found: %s", err)
			continue
		}
		if _, err := nrpc.Request(sfu, proto.ToSfuLeaveMsg{
			MID: mid,
		}); err != nil {
			log.Errorf("SfuClientLeave error: %v", err.Error())
			continue
		}
	}

	return nil
}

// Broadcast peer send message to peers of room
func (p *peer) broadcast(msg *proto.FromClientBroadcastMsg) error {
	log.Infof("peer broadcast: uid=%s, msg=%v", p.id, msg)

	// Validate
	if msg.RID == "" {
		return errors.New("room not found")
	}

	islb := getIslb()
	if islb == "" {
		return errors.New("islb-node not found")
	}

	// TODO: nrpc.Publish(roomID, ...
	err := nrpc.Publish(islb, proto.IslbBroadcastMsg{
		RoomInfo: proto.RoomInfo{UID: p.id, RID: msg.RID},
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

// request send msg to message and waits for the response
func (p *peer) request(method string, params, result interface{}) error {
	return p.conn.Call(p.ctx, method, params, result)
}

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
	p.roomLook.Lock()
	rooms := p.rooms
	p.roomLook.Unlock()
	for rid, mid := range rooms {
		p.leave(&proto.FromClientLeaveMsg{
			UID: p.id,
			RID: rid,
			MID: mid,
		})
	}

	if p.onCloseFun != nil {
		p.onCloseFun()
	}
}

// setCloseFun sets a handler that is called when the peer close
func (p *peer) setCloseFun(f func()) {
	p.onCloseFun = f
}
