package biz

import (
	"errors"

	"github.com/google/uuid"
	"github.com/notedit/sdp"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/signal"
)

var (
	ridError  = newError(codeRoomErr, codeStr(codeRoomErr))
	jsepError = newError(codeJsepErr, codeStr(codeJsepErr))
	sdpError  = newError(codeSDPErr, codeStr(codeSDPErr))
	midError  = newError(codeMIDErr, codeStr(codeMIDErr))
)

// join room
func join(peer *signal.Peer, msg proto.FromClientJoinMsg) (interface{}, *httpError) {
	log.Infof("biz.join peer.ID()=%s msg=%v", peer.ID(), msg)
	rid := msg.RID

	// validate
	if msg.RID == "" {
		return nil, ridError
	}
	sdpInfo, err := sdp.Parse(msg.Jsep.SDP)
	if err != nil {
		return nil, sdpError
	}

	islb := getIslb()
	if islb == "" {
		return nil, newError(500, "Not found any node for islb.")
	}
	uid := peer.ID()

	// already joined this room, removing old peer
	if p := signal.GetPeer(rid, uid); p != nil {
		log.Infof("biz.join peer.ID()=%s already joined, removing old peer", uid)
		if _, err := nrpc.Request(islb, proto.IslbPeerLeaveMsg{
			RoomInfo: proto.RoomInfo{UID: uid, RID: msg.RID},
		}); err != nil {
			log.Errorf("IslbClientOnLeave failed %v", err.Error())
		}
		p.Close()
	}
	log.Infof("biz.join adding new peer")
	signal.AddPeer(rid, peer)

	mid := proto.MID(uuid.New().String())
	sfu, err := getNode("sfu", islb, uid, rid, mid)
	if err != nil {
		log.Errorf("getting sfu-node: %v", err)
		return nil, newError(500, "Not found any node for sfu.")
	}
	info := msg.Info

	// join to islb
	resp, err := nrpc.Request(islb, proto.ToIslbPeerJoinMsg{
		UID: uid, RID: rid, MID: mid, Info: info,
	})
	if err != nil {
		log.Errorf("IslbClientOnJoin failed %v", err)
	}
	fromIslbPeerJoinMsg, ok := resp.(*proto.FromIslbPeerJoinMsg)
	if !ok {
		log.Errorf("IslbClientOnJoin failed %v", fromIslbPeerJoinMsg)
	}

	// handle sfu message
	rpcID := nid + "-" + string(rid) + "-" + string(uid)
	sub, err := nrpc.Subscribe(rpcID, func(msg interface{}) (interface{}, error) {
		log.Infof("peer(%s) handle sfu message: %+v", uid, msg)
		switch v := msg.(type) {
		case *proto.SfuTrickleMsg:
			log.Infof("peer(%s) got a remote candidate: %s", uid, v.Candidate)
			signal.NotifyPeer(proto.ClientTrickleICE, rid, uid, proto.ClientTrickleMsg{
				RID:       rid,
				MID:       v.MID,
				Candidate: proto.CandidateForJSON(v.Candidate),
			})
		case *proto.SfuOfferMsg:
			log.Infof("peer(%s) got remote description: %s", uid, v.Jsep)
			signal.NotifyPeer(proto.ClientOffer, rid, uid, proto.ClientOfferMsg{
				RID:     rid,
				MID:     v.MID,
				RTCInfo: v.RTCInfo,
			})
		default:
			return nil, errors.New("unkonw message")
		}
		return nil, nil
	})
	if err != nil {
		log.Errorf("subscribe sfu failed: %v", err)
		return nil, newError(500, "subscribe sfu failed")
	}
	peer.SetCloseFun(func() {
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
		log.Errorf("join sfu error: %v", err)
	}
	fromSfuJoinMsg, ok := resp.(*proto.FromSfuJoinMsg)
	if !ok {
		log.Errorf("join reply msg parses failed")
		return nil, newError(500, "join reply msg parses failed")
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

func leave(peer *signal.Peer, msg proto.FromClientLeaveMsg) (interface{}, *httpError) {
	log.Infof("biz.leave msg=%v", msg)
	room := signal.GetRoom(msg.RID)
	if room == nil {
		log.Warnf("room not exits, rid=", msg.RID)
		return nil, nil
	}
	room.DelPeer(msg.UID)
	peer.Close()

	islb := getIslb()
	if islb == "" {
		log.Errorf("islb node not found")
		return nil, newError(500, "islb node not found")
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
			log.Errorf("Not found any sfu node: %s", err)
			continue
		}
		if _, err := nrpc.Request(sfu, proto.ToSfuLeaveMsg{
			MID: mid,
		}); err != nil {
			log.Errorf("SfuClientLeave error: %v", err.Error())
			continue
		}
	}

	return nil, nil
}

func offer(peer *signal.Peer, msg proto.ClientOfferMsg) (interface{}, *httpError) {
	log.Infof("biz.offer peer.ID()=%s msg=%v", peer.ID(), msg)

	sfu, err := getNode("sfu", "", peer.ID(), msg.RID, msg.MID)
	if err != nil {
		log.Warnf("Not found any sfu node: %s", err)
		return nil, newError(500, "Not found any sfu node")
	}

	resp, err := nrpc.Request(sfu, proto.SfuOfferMsg{
		MID:     msg.MID,
		RTCInfo: proto.RTCInfo{Jsep: msg.Jsep},
	})
	if err != nil {
		log.Errorf("SfuClientOnOffer failed %v", err.Error())
		return nil, newError(500, "SfuClientOnOffer failed")
	}

	answer, ok := resp.(*proto.SfuAnswerMsg)
	if !ok {
		log.Errorf("Parse answer failed %v", err.Error())
		return nil, newError(500, "Parse answer failed")
	}

	return proto.ClientAnswerMsg{
		RID:     msg.RID,
		MID:     msg.MID,
		RTCInfo: answer.RTCInfo,
	}, nil
}

func answer(peer *signal.Peer, msg proto.ClientAnswerMsg) (interface{}, *httpError) {
	log.Infof("biz.answer peer.ID()=%s msg=%v", peer.ID(), msg)

	sfu, err := getNode("sfu", "", peer.ID(), msg.RID, msg.MID)
	if err != nil {
		log.Warnf("Not found any sfu node: %s", err)
		return nil, newError(500, "Not found any sfu node")
	}

	if _, err := nrpc.Request(sfu, proto.SfuAnswerMsg{
		MID:     msg.MID,
		RTCInfo: msg.RTCInfo,
	}); err != nil {
		log.Errorf("SfuClientOnAnswer failed %v", err.Error())
		return nil, newError(500, err.Error())
	}

	return emptyMap, nil
}

func broadcast(peer *signal.Peer, msg proto.FromClientBroadcastMsg) (interface{}, *httpError) {
	log.Infof("biz.broadcast peer.ID()=%s msg=%v", peer.ID(), msg)

	// Validate
	if msg.RID == "" {
		return nil, ridError
	}

	islb := getIslb()
	if islb == "" {
		return nil, newError(500, "Not found any node for islb.")
	}

	err := nrpc.Publish(islb, proto.IslbBroadcastMsg{
		RoomInfo: proto.RoomInfo{UID: peer.ID(), RID: msg.RID},
		Info:     msg.Info,
	})
	if err != nil {
		log.Errorf("Broadcast error: %s", err.Error())
		return nil, newError(500, "Broadcast error")
	}

	return emptyMap, nil
}

func trickle(peer *signal.Peer, msg proto.ClientTrickleMsg) (interface{}, *httpError) {
	log.Infof("biz.trickle peer.ID()=%s msg=%v", peer.ID(), msg)
	if msg.RID == "" {
		return nil, ridError
	}

	sfu, err := getNode("sfu", "", peer.ID(), msg.RID, msg.MID)
	if err != nil {
		log.Warnf("Not found any sfu node: %s", err.Error())
		return nil, newError(500, "Not found any sfu node")
	}

	err = nrpc.Publish(sfu, proto.SfuTrickleMsg{
		MID:       msg.MID,
		Candidate: msg.Candidate,
	})
	if err != nil {
		log.Errorf("Send trickle to sfu error: %s", err.Error())
		return nil, newError(500, "Send trickle to sfu error")
	}

	return emptyMap, nil
}
