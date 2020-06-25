package biz

import (
	"context"
	"fmt"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion-sfu/pkg/proto/media"
	"github.com/pion/ion-sfu/pkg/proto/sfu"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/signal"
	"github.com/pion/ion/pkg/util"
)

var (
	ridError  = util.NewNpError(codeRoomErr, codeStr(codeRoomErr))
	jsepError = util.NewNpError(codeJsepErr, codeStr(codeJsepErr))
	// sdpError  = util.NewNpError(codeSDPErr, codeStr(codeSDPErr))
	midError = util.NewNpError(codeMIDErr, codeStr(codeMIDErr))
)

// join room
func join(peer *signal.Peer, msg proto.JoinMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.join peer.ID()=%s msg=%v", peer.ID(), msg)
	rid := msg.RID

	// Validate
	if msg.RID == "" {
		return nil, ridError
	}

	//already joined this room
	if signal.HasPeer(rid, peer) {
		return emptyMap, nil
	}
	signal.AddPeer(rid, peer)

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}
	// Send join => islb
	info := msg.Info
	uid := peer.ID()
	_, err := islb.SyncRequest(proto.IslbClientOnJoin, util.Map("rid", rid, "uid", uid, "info", info))
	if err != nil {
		log.Errorf("IslbClientOnJoin failed %v", err.Error())
	}
	// Send getPubs => islb
	islb.AsyncRequest(proto.IslbGetPubs, msg.RoomInfo).Then(
		func(result nprotoo.RawMessage) {
			var resMsg proto.GetPubResp
			if err := result.Unmarshal(&resMsg); err != nil {
				log.Errorf("Unmarshal pub response %v", err)
				return
			}
			log.Infof("IslbGetPubs: result=%v", result)
			for _, pub := range resMsg.Pubs {
				if pub.MID == "" {
					continue
				}
				notif := proto.StreamAddMsg(pub)
				peer.Notify(proto.ClientOnStreamAdd, notif)
			}
		},
		func(err *nprotoo.Error) {})

	return emptyMap, nil
}

func leave(peer *signal.Peer, msg proto.LeaveMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.leave peer.ID()=%s msg=%v", peer.ID(), msg)
	defer util.Recover("biz.leave")

	rid := msg.RID

	// Validate
	if msg.RID == "" {
		return nil, ridError
	}

	uid := peer.ID()

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}

	islb.AsyncRequest(proto.IslbOnStreamRemove, util.Map("rid", rid, "uid", uid))
	_, err := islb.SyncRequest(proto.IslbClientOnLeave, util.Map("rid", rid, "uid", uid))
	if err != nil {
		log.Errorf("IslbOnStreamRemove failed %v", err.Error())
	}
	signal.DelPeer(rid, peer.ID())
	return emptyMap, nil
}

func publish(peer *signal.Peer, msg proto.PublishMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.publish peer.ID()=%s", peer.ID())

	nid, sfuClient, err := getRPCForSFU("")
	if err != nil {
		log.Warnf("sfu node not found, reject: %s", err)
		return nil, util.NewNpError(500, fmt.Sprintf("%s", err))
	}

	jsep := msg.Jsep
	options := msg.Options
	room := signal.GetRoomByPeer(peer.ID())
	if room == nil {
		return nil, util.NewNpError(codeRoomErr, codeStr(codeRoomErr))
	}

	rid := room.ID()
	uid := peer.ID()

	stream, err := sfuClient.Publish(context.Background(), &sfu.PublishRequest{
		Rid: "default",
		Options: &sfu.PublishOptions{
			Codec:       options.Codec,
			Bandwidth:   uint32(options.Bandwidth),
			Transportcc: options.TransportCC,
		},
		Description: &sfu.SessionDescription{
			Type: jsep.Type.String(),
			Sdp:  jsep.SDP,
		},
	})

	if err != nil {
		log.Warnf("reject: %s", err)
		return nil, util.NewNpError(500, fmt.Sprintf("%s", err))
	}

	answer, err := stream.Recv()

	log.Infof("publish: result => %v", answer)
	mid := answer.Mediainfo.Mid
	tracks := answer.Stream.Tracks

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}
	islb.AsyncRequest(proto.IslbOnStreamAdd, util.Map("rid", rid, "nid", nid, "uid", uid, "mid", mid, "tracks", tracks))

	go func() {
		// Next response is received on webrtc transport close
		answer, err = stream.Recv()
		log.Infof("Pub closed => %s", mid)
		islb.AsyncRequest(proto.IslbOnStreamRemove, util.Map("rid", rid, "nid", nid, "uid", uid, "mid", mid))
	}()

	return answer, nil
}

func subscribe(peer *signal.Peer, msg proto.SubscribeMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.subscribe peer.ID()=%s ", peer.ID())
	mid := msg.MID

	// Validate
	if mid == "" {
		return nil, midError
	} else if msg.Jsep.SDP == "" {
		return nil, jsepError
	}

	nodeID, sfuClient, err := getRPCForSFU(string(mid))
	if err != nil {
		log.Warnf("sfu node not found, reject: %s", err)
		return nil, util.NewNpError(500, fmt.Sprintf("%s", err))
	}

	// TODO:
	if nodeID != "node for mid" {
		log.Warnf("Not the same node, need to enable sfu-sfu relay!")
	}

	room := signal.GetRoomByPeer(peer.ID())
	rid := room.ID()

	jsep := msg.Jsep

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}

	result, nerr := islb.SyncRequest(proto.IslbGetMediaInfo, proto.MediaInfo{RID: rid, MID: mid})
	if err != nil {
		log.Warnf("reject: %d => %s", nerr.Code, nerr.Reason)
		return nil, util.NewNpError(nerr.Code, nerr.Reason)
	}
	var some map[string]map[string][]proto.TrackInfo
	if err := result.Unmarshal(&some); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	for msid, tracks := range some["tracks"] {
		var mtracks []*media.Track
		for _, track := range tracks {
			mtrack := media.Track{
				Id:      track.ID,
				Ssrc:    uint32(track.Ssrc),
				Payload: uint32(track.Payload),
				Type:    track.Type,
				Codec:   track.Codec,
				Fmtp:    track.Fmtp,
			}
			mtracks = append(mtracks, &mtrack)
		}

		answer, err := sfuClient.Subscribe(ctx, &sfu.SubscribeRequest{
			Mid: string(msg.MID),
			Description: &sfu.SessionDescription{
				Type: jsep.Type.String(),
				Sdp:  jsep.SDP,
			},
			Stream: &media.Stream{
				Id:     msid,
				Tracks: mtracks,
			},
		})

		if err != nil {
			return nil, util.NewNpError(500, "error subscribing to stream")
		}

		log.Infof("subscribe: result => %v", result)
		return answer, nil
	}

	return nil, util.NewNpError(404, "stream not found")
}

func broadcast(peer *signal.Peer, msg proto.BroadcastMsg) (interface{}, *nprotoo.Error) {
	log.Infof("biz.broadcast peer.ID()=%s msg=%v", peer.ID(), msg)

	// Validate
	if msg.RID == "" || msg.UID == "" {
		return nil, ridError
	}

	islb, found := getRPCForIslb()
	if !found {
		return nil, util.NewNpError(500, "Not found any node for islb.")
	}
	rid, uid, info := msg.RID, msg.UID, msg.Info
	islb.AsyncRequest(proto.IslbOnBroadcast, util.Map("rid", rid, "uid", uid, "info", info))
	return emptyMap, nil
}

func trickle(peer *signal.Peer, msg proto.TrickleMsg) (interface{}, *nprotoo.Error) {
	// log.Infof("biz.trickle peer.ID()=%s msg=%v", peer.ID(), msg)
	// mid := msg.MID

	// // Validate
	// if msg.RID == "" || msg.UID == "" {
	// 	return nil, ridError
	// }

	// _, sfu, err := getRPCForSFU(mid)
	// if err != nil {
	// 	log.Warnf("Not found any sfu node, reject: %d => %s", err.Code, err.Reason)
	// 	return nil, util.NewNpError(err.Code, err.Reason)
	// }

	// trickle := msg.Trickle

	// sfu.AsyncRequest(proto.ClientTrickleICE, util.Map("mid", mid, "trickle", trickle))
	return emptyMap, nil
}
