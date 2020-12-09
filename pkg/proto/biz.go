package proto

import (
	"encoding/gob"
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

func init() {
	gob.Register(&FromClientJoinMsg{})
	gob.Register(&ToClientJoinMsg{})
	gob.Register(&ToClientPeersMsg{})
	gob.Register(&ToClientPeerJoinMsg{})
	gob.Register(&ClientOfferMsg{})
	gob.Register(&ClientAnswerMsg{})
	gob.Register(&ClientTrickleMsg{})
	gob.Register(&FromClientLeaveMsg{})
	gob.Register(&FromClientBroadcastMsg{})
	gob.Register(&ToClientBroadcastMsg{})

	gob.Register(&ToSfuJoinMsg{})
	gob.Register(&FromSfuJoinMsg{})
	gob.Register(&ToSfuLeaveMsg{})
	gob.Register(&SfuTrickleMsg{})
	gob.Register(&SfuOfferMsg{})
	gob.Register(&SfuAnswerMsg{})
	gob.Register(&SfuICEConnectionStateMsg{})

	gob.Register(&ToAvpProcessMsg{})

	gob.Register(&IslbBroadcastMsg{})
	gob.Register(&ToIslbPeerJoinMsg{})
	gob.Register(&FromIslbPeerJoinMsg{})
	gob.Register(&IslbPeerLeaveMsg{})
	gob.Register(&ToIslbStreamAddMsg{})
	gob.Register(&FromIslbStreamAddMsg{})
	gob.Register(&ToIslbFindNodeMsg{})
	gob.Register(&FromIslbFindNodeMsg{})
	gob.Register(&FromIslbListMids{})
}

type Authenticatable interface {
	Room() SID
	Token() string
}

type RoomToken struct {
	Token string `json:"token,omitempty"`
}

type RoomInfo struct {
	SID SID `json:"sid"`
	UID UID `json:"uid"`
}
type Peer struct {
	UID  UID             `json:"uid"`
	Info json.RawMessage `json:"info"`
}

type Stream struct {
	StreamID StreamID `json:"streamId"`
	UID      UID      `json:"uid"`
}

type TrackMap map[string][]TrackInfo

// Client <-> Biz messages.

type FromClientJoinMsg struct {
	SID   SID                       `json:"sid"`
	Offer webrtc.SessionDescription `json:"offer"`
	RoomToken
	Info json.RawMessage `json:"info"`
}

func (j *FromClientJoinMsg) Token() string {
	return j.RoomToken.Token
}
func (j *FromClientJoinMsg) Room() SID {
	return j.SID
}

type ToClientJoinMsg struct {
	Answer webrtc.SessionDescription `json:"answer"`
}

type ToClientPeersMsg struct {
	Peers   []Peer   `json:"peers"`
	Streams []Stream `json:"streams"`
}

type ToClientPeerJoinMsg struct {
	UID  UID             `json:"uid"`
	SID  SID             `json:"sid"`
	Info json.RawMessage `json:"info"`
}

type ClientOfferMsg struct {
	Desc webrtc.SessionDescription `json:"desc"`
}

type ClientAnswerMsg struct {
	Desc webrtc.SessionDescription `json:"desc"`
}

type ClientTrickleMsg struct {
	Candidate webrtc.ICECandidateInit `json:"candidate"`
	Target    int                     `json:"target"`
}

type FromClientLeaveMsg struct {
	SID SID `json:"sid"`
}

type FromClientBroadcastMsg struct {
	SID  SID             `json:"sid"`
	Info json.RawMessage `json:"info"`
}

type ToClientBroadcastMsg struct {
	RoomInfo
	Info json.RawMessage `json:"info"`
}

// Biz to SFU

type ToSfuJoinMsg struct {
	RPC   string                    `json:"rpc"`
	SID   SID                       `json:"sid"`
	UID   UID                       `json:"uid"`
	MID   MID                       `json:"mid"`
	Offer webrtc.SessionDescription `json:"offer"`
}

type FromSfuJoinMsg struct {
	Answer webrtc.SessionDescription `json:"answer"`
}

type ToSfuLeaveMsg struct {
	MID MID `json:"mid"`
}

type SfuTrickleMsg struct {
	SID       SID                     `json:"sid"`
	UID       UID                     `json:"uid"`
	MID       MID                     `json:"mid"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
	Target    int                     `json:"target"`
}

type SfuOfferMsg struct {
	SID  SID                       `json:"sid"`
	UID  UID                       `json:"uid"`
	MID  MID                       `json:"mid"`
	Desc webrtc.SessionDescription `json:"offer"`
}

type SfuAnswerMsg struct {
	MID  MID                       `json:"mid"`
	Desc webrtc.SessionDescription `json:"answer"`
}

type SfuICEConnectionStateMsg struct {
	SID   SID                       `json:"sid"`
	UID   UID                       `json:"uid"`
	MID   MID                       `json:"mid"`
	State webrtc.ICEConnectionState `json:"state"`
}

// Biz to AVP

// ToAvpProcessMsg .
type ToAvpProcessMsg struct {
	Addr   string `json:"Addr"`
	PID    string `json:"pid"`
	SID    string `json:"sid"`
	TID    string `json:"tid"`
	EID    string `json:"eid"`
	Config []byte `json:"config"`
}

// Islb messages

type IslbBroadcastMsg struct {
	RoomInfo
	Info json.RawMessage `json:"info"`
}

type ToIslbPeerJoinMsg struct {
	UID  UID             `json:"uid"`
	SID  SID             `json:"sid"`
	MID  MID             `json:"mid"`
	Info json.RawMessage `json:"info"`
}

type FromIslbPeerJoinMsg struct {
	Peers   []Peer   `json:"peers"`
	Streams []Stream `json:"streams"`
}

type IslbPeerLeaveMsg struct {
	RoomInfo
}

type StreamID string

type ToIslbStreamAddMsg struct {
	UID      UID      `json:"uid"`
	SID      SID      `json:"sid"`
	MID      MID      `json:"mid"`
	StreamID StreamID `json:"streamId"`
}

type FromIslbStreamAddMsg struct {
	UID    UID    `json:"uid"`
	SID    SID    `json:"sid"`
	Stream Stream `json:"stream"`
}

type ToIslbFindNodeMsg struct {
	Service string
	UID     UID `json:"uid"`
	SID     SID `json:"sid"`
	MID     MID `json:"mid"`
}

type FromIslbFindNodeMsg struct {
	ID string
}

type FromIslbListMids struct {
	MIDs []MID `json:"mids"`
}

// CandidateForJSON for json.Marshal() => browser
func CandidateForJSON(c webrtc.ICECandidateInit) webrtc.ICECandidateInit {
	if c.SDPMid == nil {
		c.SDPMid = refString("0")
	}
	if c.SDPMLineIndex == nil {
		c.SDPMLineIndex = refUint16(0)
	}
	return c
}

func refString(s string) *string {
	return &s
}

func refUint16(i uint16) *uint16 {
	return &i
}
