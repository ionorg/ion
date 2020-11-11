package biz

import (
	"errors"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/signal"
)

func handleIslbBroadcast(msg interface{}) (interface{}, error) {
	go func(msg interface{}) {
		log.Infof("handle islb message: %v", msg)

		var method string
		var rid proto.RID
		var uid proto.UID

		switch v := msg.(type) {
		case *proto.FromIslbStreamAddMsg:
			method, rid, uid = proto.ClientOnStreamAdd, v.RID, v.UID
		case *proto.ToClientPeerJoinMsg:
			method, rid, uid = proto.ClientOnJoin, v.RID, v.UID
		case *proto.IslbPeerLeaveMsg:
			method, rid, uid = proto.ClientOnLeave, v.RID, v.UID
		case *proto.IslbBroadcastMsg:
			method, rid, uid = proto.ClientBroadcast, v.RID, v.UID
		default:
			log.Warnf("unkonw message: %v", msg)
		}

		log.Infof("broadcast: method=%s, msg=%v", method, msg)
		if r := signal.GetRoom(rid); r != nil {
			r.NotifyWithoutID(method, msg, uid)
		} else {
			log.Warnf("room not exits, rid=%s, uid=%", rid, uid)
		}

	}(msg)

	return nil, nil
}

func getIslb() string {
	nodes := getNodes()
	for _, item := range nodes {
		if item.Service == "islb" {
			return item.NID
		}
	}
	log.Warnf("No islb node was found.")
	return ""
}

func getNode(service string, islb string, uid proto.UID, rid proto.RID, mid proto.MID) (string, error) {
	if islb == "" {
		if islb = getIslb(); islb == "" {
			return "", errors.New("Not found islb")
		}
	}

	resp, err := nrpc.Request(islb, proto.ToIslbFindNodeMsg{
		Service: service,
		UID:     uid,
		RID:     rid,
		MID:     mid,
	})

	if err != nil {
		return "", err
	}

	msg, ok := resp.(*proto.FromIslbFindNodeMsg)
	if !ok {
		return "", errors.New("parse islb-find-node msg error")
	}

	return msg.ID, nil
}
