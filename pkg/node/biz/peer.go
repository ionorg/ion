package biz

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/notedit/sdp"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/webrtc/v3"
)

// Peer represents a peer for client
type Peer struct {
	uid       proto.UID
	mid       proto.MID
	sid       proto.SID
	info      []byte
	ctx       context.Context
	leaveOnce sync.Once
	closed    util.AtomicBool
	s         *Server
	send      func(msg interface{}) error
}

// NewPeer create peer instance for client
func NewPeer(uid proto.UID, s *Server, send func(msg interface{}) error) *Peer {
	id := uuid.New().String()
	p := &Peer{
		uid:  uid,
		mid:  proto.MID(id),
		s:    s,
		send: send,
	}
	return p
}

// handleSFUMessage handle sfu message
func (p *Peer) handleSFUMessage(msg interface{}) {
	switch v := msg.(type) {
	case *proto.SfuOfferMsg:
		log.Infof("peer(%s) got remote description: %v", p.uid, v.Desc)
		if err := p.send(&proto.ClientOfferMsg{
			Desc: v.Desc,
		}); err != nil {
			log.Errorf("error sending offer %s", err)
		}
	case *proto.SfuTrickleMsg:
		log.Infof("peer(%s) got a remote candidate: %v", p.uid, v.Candidate)
		if err := p.send(&proto.ClientTrickleMsg{
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
				if err := p.Leave(&proto.FromClientLeaveMsg{SID: p.sid}); err != nil {
					log.Infof("peer(%s) leave error: %v", p.sid, err)
				}
			})
		}
	}
}

// Join client Join the room
func (p *Peer) Join(msg *proto.FromClientJoinMsg) (interface{}, error) {
	log.Infof("peer join: uid=%s, msg=%v", p.uid, msg)

	p.sid = msg.SID
	p.info = msg.Info

	// validate
	if p.sid == "" {
		return nil, errors.New("room not found")
	}

	// join room
	p.s.addPeer(p.sid, p)
	log.Errorf("[%s] join to room %s", p.uid, p.sid)

	// get sfu node
	sfu, err := p.sfu()
	if err != nil {
		p.s.delPeer(p.sid, p.uid)
		log.Errorf("[%s] sfu not found: %v", p.uid, err)
		return nil, errors.New("sfu not found")
	}

	// join to sfu
	resp, err := p.s.nrpc.Request(sfu, proto.ToSfuJoinMsg{
		RPC:   p.s.nid,
		MID:   p.mid,
		UID:   p.uid,
		SID:   p.sid,
		Offer: msg.Offer,
	})
	if err != nil {
		p.s.delPeer(p.sid, p.uid)
		log.Errorf("[%s] join to %s error: %v", p.uid, sfu, err)
		return nil, errors.New("join to sfu error")
	}
	fromSfuJoinMsg := resp.(*proto.FromSfuJoinMsg)

	// join to islb
	resp, err = p.s.nrpc.Request(p.s.islb, proto.ToIslbPeerJoinMsg{
		UID: p.uid, SID: p.sid, MID: p.mid, Info: p.info,
	})
	if err != nil {
		if _, err := p.s.nrpc.Request(sfu, proto.ToSfuLeaveMsg{
			MID: p.mid,
		}); err != nil {
			log.Errorf("[%s] leave %s error: %v", p.uid, sfu, err.Error())
		}
		p.s.delPeer(p.sid, p.uid)
		log.Errorf("[%s] join to %s error: %v", p.uid, p.s.islb, err)
		return nil, errors.New("join to sfu error")
	}

	// send peer-list to clients
	fromIslbPeerJoinMsg := resp.(*proto.FromIslbPeerJoinMsg)
	go func(peerlist proto.ToClientPeersMsg) {
		time.Sleep(100 * time.Millisecond)
		if err := p.send(&peerlist); err != nil {
			log.Errorf("[%s] send peer-list to clients error: %v", p.uid, err)
		}
	}(proto.ToClientPeersMsg{
		Peers:   fromIslbPeerJoinMsg.Peers,
		Streams: fromIslbPeerJoinMsg.Streams,
	})

	return fromSfuJoinMsg.Answer, nil
}

// Offer client send Offer to biz
func (p *Peer) Offer(msg *proto.ClientOfferMsg) (interface{}, error) {
	log.Infof("peer offer: uid=%s, msg=%v", p.uid, msg)

	// send offer to sfu
	sfu, err := p.sfu()
	if err != nil {
		return nil, err
	}
	resp, err := p.s.nrpc.Request(sfu, proto.SfuOfferMsg{
		MID:  p.mid,
		Desc: msg.Desc,
	})
	if err != nil {
		log.Errorf("offer %s failed %v", sfu, err.Error())
		return nil, err
	}

	// associate the stream in the SDP with the UID/SID/MID.
	sdpInfo, err := sdp.Parse(msg.Desc.SDP)
	if err != nil {
		log.Errorf("parse sdp error: %v", err)
	}
	for key := range sdpInfo.GetStreams() {
		if err := p.s.nrpc.Publish(p.s.islb, proto.ToIslbStreamAddMsg{
			UID: p.uid, SID: p.sid, MID: p.mid, StreamID: proto.StreamID(key),
		}); err != nil {
			log.Errorf("send stream-add to %s error: %v", p.s.islb, err)
		}
	}

	// avp sub streams
	var avp string
	if len(p.s.elements) > 0 {
		if avp, err = p.avp(); err != nil {
			log.Errorf("get avp-node error: %v", err)
		}
	}
	if avp != "" {
		if err != nil {
			log.Errorf("parse sdp error: %v", err)
		}
		for _, eid := range p.s.elements {
			for _, stream := range sdpInfo.GetStreams() {
				tracks := stream.GetTracks()
				for _, track := range tracks {
					err = p.s.nrpc.Publish(avp, proto.ToAvpProcessMsg{
						Addr:   sfu,
						PID:    stream.GetID(),
						SID:    string(p.sid),
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

	return resp.(*proto.SfuAnswerMsg).Desc, nil
}

// Answer received Answer of client
func (p *Peer) Answer(msg *proto.ClientAnswerMsg) error {
	log.Infof("peer answer:  uid=%s, msg=%v", p.uid, msg)

	sfu, err := p.sfu()
	if err != nil {
		log.Warnf("sfu-node not found, %s", err.Error())
		return err
	}

	if _, err := p.s.nrpc.Request(sfu, proto.SfuAnswerMsg{
		MID:  p.mid,
		Desc: msg.Desc,
	}); err != nil {
		log.Errorf("answer %s error: %v", sfu, err.Error())
		return err
	}

	return nil
}

// Trickle received candidate of client
func (p *Peer) Trickle(msg *proto.ClientTrickleMsg) error {
	log.Infof("peer trickle: uid=%s, msg=%v", p.uid, msg)

	sfu, err := p.sfu()
	if err != nil {
		log.Warnf("sfu-node not found, %s", err.Error())
		return err
	}

	_, err = p.s.nrpc.Request(sfu, proto.SfuTrickleMsg{
		MID:       p.mid,
		Candidate: msg.Candidate,
		Target:    msg.Target,
	})
	if err != nil {
		log.Errorf("trickle to %s error: %s", sfu, err.Error())
		return err
	}

	return nil
}

// Leave client leave the room
func (p *Peer) Leave(msg *proto.FromClientLeaveMsg) error {
	log.Infof("peer leave: uid=%s, msg=%v", p.uid, msg)

	// leave room
	p.s.delPeer(msg.SID, p.uid)

	if _, err := p.s.nrpc.Request(p.s.islb, proto.IslbPeerLeaveMsg{
		RoomInfo: proto.RoomInfo{UID: p.uid, SID: msg.SID},
	}); err != nil {
		log.Errorf("leave %s error: %v", p.s.islb, err.Error())
	}

	sfu, err := p.sfu()
	if err != nil {
		log.Errorf("sfu-node not found: %s", err)
	}
	if _, err := p.s.nrpc.Request(sfu, proto.ToSfuLeaveMsg{
		MID: p.mid,
	}); err != nil {
		log.Errorf("leave %s error: %v", sfu, err.Error())
	}

	return nil
}

// Broadcast peer send message to peers of room
func (p *Peer) Broadcast(msg *proto.FromClientBroadcastMsg) error {
	log.Infof("peer broadcast: uid=%s, msg=%v", p.uid, msg)

	// validate
	if msg.SID == "" {
		return errors.New("room not found")
	}

	// TODO: nrpc.Publish(roomID, ...
	err := p.s.nrpc.Publish(p.s.islb, proto.IslbBroadcastMsg{
		RoomInfo: proto.RoomInfo{UID: p.uid, SID: msg.SID},
		Info:     msg.Info,
	})
	if err != nil {
		log.Errorf("broadcast error: %s", err.Error())
		return err
	}

	return nil
}

// Close peer
func (p *Peer) Close() {
	if p.closed.Get() {
		return
	}
	p.closed.Set(true)

	// leave all rooms
	if err := p.Leave(&proto.FromClientLeaveMsg{SID: p.sid}); err != nil {
		log.Infof("peer(%s) leave error: %v", p.sid, err)
	}
}

// UID return peer uid
func (p *Peer) UID() proto.UID {
	return p.uid
}

func (p *Peer) sfu() (string, error) {
	return p.s.getNode(proto.ServiceSFU, p.uid, p.sid, p.mid)
}

func (p *Peer) avp() (string, error) {
	return p.s.getNode(proto.ServiceAVP, p.uid, p.sid, p.mid)
}
