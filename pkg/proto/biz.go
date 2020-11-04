package proto

import (
	"encoding/gob"
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

func init() {
	gob.Register(&FromClientJoinMsg{})
	gob.Register(&ToClientJoinMsg{})
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

	gob.Register(&ToAvpProcessMsg{})

	gob.Register(&IslbBroadcastMsg{})
	gob.Register(&ToIslbPeerJoinMsg{})
	gob.Register(&FromIslbPeerJoinMsg{})
	gob.Register(&IslbPeerLeaveMsg{})
	gob.Register(&ToIslbStreamAddMsg{})
	gob.Register(&FromIslbStreamAddMsg{})
	gob.Register(&ToIslbFindNodeMsg{})
	gob.Register(&FromIslbFindNodeMsg{})
	gob.Register(&ToIslbListMids{})
	gob.Register(&FromIslbListMids{})
}

type Authenticatable interface {
	Room() RID
	Token() string
}

type RoomToken struct {
	Token string `json:"token,omitempty"`
}

type RoomInfo struct {
	RID RID `json:"rid"`
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

type RTCInfo struct {
	Jsep webrtc.SessionDescription `json:"jsep"`
}

type TrackMap map[string][]TrackInfo

// Client <-> Biz messages.

type FromClientJoinMsg struct {
	RID RID `json:"rid"`
	RoomToken
	RTCInfo
	Info json.RawMessage `json:"info"`
}

func (j *FromClientJoinMsg) Token() string {
	return j.RoomToken.Token
}
func (j *FromClientJoinMsg) Room() RID {
	return j.RID
}

type ToClientJoinMsg struct {
	Peers   []Peer   `json:"peers"`
	Streams []Stream `json:"streams"`
	MID     MID      `json:"mid"`
	RTCInfo
}

type ToClientPeerJoinMsg struct {
	UID  UID             `json:"uid"`
	RID  RID             `json:"rid"`
	Info json.RawMessage `json:"info"`
}

type ClientOfferMsg struct {
	RID RID `json:"rid"`
	MID MID `json:"mid"`
	RTCInfo
}

type ClientAnswerMsg struct {
	RID RID `json:"rid"`
	MID MID `json:"mid"`
	RTCInfo
}

type ClientTrickleMsg struct {
	RID       RID                     `json:"rid"`
	MID       MID                     `json:"mid"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type FromClientLeaveMsg struct {
	UID UID `json:"uid"`
	RID RID `json:"rid"`
	MID MID `json:"mid"`
}

type FromClientBroadcastMsg struct {
	RID  RID             `json:"rid"`
	Info json.RawMessage `json:"info"`
}

type ToClientBroadcastMsg struct {
	RoomInfo
	Info json.RawMessage `json:"info"`
}

// Biz to SFU

type ToSfuJoinMsg struct {
	RPCID string `json:"rpc"`
	RID   RID    `json:"rid"`
	MID   MID    `json:"mid"`
	RTCInfo
}

type FromSfuJoinMsg struct {
	RTCInfo
}

type ToSfuLeaveMsg struct {
	MID MID `json:"mid"`
}

type SfuTrickleMsg struct {
	MID       MID                     `json:"mid"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type SfuOfferMsg struct {
	MID MID `json:"mid"`
	RTCInfo
}

type SfuAnswerMsg struct {
	MID MID `json:"mid"`
	RTCInfo
}

// Biz to AVP

// ToAvpProcessMsg .
type ToAvpProcessMsg struct {
	Addr   string `json:"Addr"`
	PID    string `json:"pid"`
	RID    string `json:"rid"`
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
	RID  RID             `json:"rid"`
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
	RID      RID      `json:"rid"`
	MID      MID      `json:"mid"`
	StreamID StreamID `json:"streamId"`
}

type FromIslbStreamAddMsg struct {
	UID    UID    `json:"uid"`
	RID    RID    `json:"rid"`
	Stream Stream `json:"stream"`
}

type ToIslbFindNodeMsg struct {
	Service string
	UID     UID `json:"uid"`
	RID     RID `json:"rid"`
	MID     MID `json:"mid"`
}

type FromIslbFindNodeMsg struct {
	ID string
}

type ToIslbListMids struct {
	UID UID `json:"uid"`
	RID RID `json:"rid"`
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
