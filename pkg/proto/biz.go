package proto

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

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

type PublishOptions struct {
	Codec       string `json:"codec"`
	Resolution  string `json:"resolution"`
	Bandwidth   int    `json:"bandwidth"`
	Audio       bool   `json:"audio"`
	Video       bool   `json:"video"`
	Screen      bool   `json:"screen"`
	TransportCC bool   `json:"transportCC,omitempty"`
	Description string `json:"description,omitempty"`
}

type SubscribeOptions struct {
	Bandwidth   int  `json:"bandwidth"`
	TransportCC bool `json:"transportCC"`
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

type ClientNegotiationMsg struct {
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
	UID UID `json:"uid"`
	RID RID `json:"rid"`
	MID MID `json:"mid"`
	SID SID `json:"sid"`
	RTCInfo
}

type FromSfuJoinMsg struct {
	RTCInfo
}

type ToSfuLeaveMsg struct {
	UID UID `json:"uid"`
	RID RID `json:"rid"`
	MID MID `json:"mid"`
}

type SfuTrickleMsg struct {
	UID       UID                     `json:"uid"`
	RID       RID                     `json:"rid"`
	MID       MID                     `json:"mid"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type SfuNegotiationMsg struct {
	UID UID `json:"uid"`
	RID RID `json:"rid"`
	MID MID `json:"mid"`
	RTCInfo
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
	SID     SID      `json:"sid"`
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
	RPCID   string
	EventID string
	ID      string
	Name    string
	Service string
}

type ToIslbListMids struct {
	UID UID `json:"uid"`
	RID RID `json:"rid"`
}

type FromIslbListMids struct {
	MIDs []MID `json:"mids"`
}

type GetSFURPCParams struct {
	RPCID   string
	EventID string
	ID      string
	Name    string
	Service string
}
