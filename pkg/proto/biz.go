package proto

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

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

<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
// TODO(kevmo314): Consolidate these messages.

=======
>>>>>>> Latest changes.
type FromClientJoinMsg struct {
	RID RID `json:"rid"`
=======
type JoinMsg struct {
<<<<<<< HEAD
	RoomInfo
>>>>>>> Handle join with ion-sfu.
=======
=======
=======
// TODO(kevmo314): Consolidate these messages.

>>>>>>> Add TODO.
type FromClientJoinMsg struct {
>>>>>>> Update SFU node to use ion-sfu.
	RID RID `json:"rid"`
>>>>>>> Add offer/answer hooks.
	RTCInfo
	Info json.RawMessage `json:"info"`
}

<<<<<<< HEAD
<<<<<<< HEAD
type ToClientJoinMsg struct {
=======
type JoinResponseMsg struct {
<<<<<<< HEAD
>>>>>>> Handle join with ion-sfu.
=======
	UID UID `json:"uid"`
>>>>>>> Add offer/answer hooks.
=======
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

type FromClientBroadcastMsg struct {
	RID  RID             `json:"rid"`
	Info json.RawMessage `json:"info"`
}

type ToClientBroadcastMsg struct {
	RoomInfo
	Info json.RawMessage `json:"info"`
}

// Signal to biz
type SignalCloseMsg struct {
	RoomInfo
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

type FromSfuLeaveMsg struct {
<<<<<<< HEAD
>>>>>>> Update SFU node to use ion-sfu.
	MediaInfo
}

type ToSfuTrickleMsg struct {
	RoomInfo
	Candidate webrtc.ICECandidateInit `json:"candidate"`
=======
	UID UID `json:"uid"`
	RID RID `json:"rid"`
	MID MID `json:"mid"`
>>>>>>> Latest changes.
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

<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
<<<<<<< HEAD
type FromSignalLeaveMsg struct {
	RoomInfo
}

type FromClientOfferMsg struct {
	RID RID `json:"rid"`
	RTCInfo
}

type ToClientOfferMsg struct {
	RID RID `json:"rid"`
	RTCInfo
}

type ToClientAnswerMsg struct {
	RID RID `json:"rid"`
	RTCInfo
}

type FromClientAnswerMsg struct {
	RID RID `json:"rid"`
	RTCInfo
}

type FromClientTrickleMsg struct {
	RID       RID                     `json:"rid"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type ToClientTrickleMsg struct {
	RID       RID                     `json:"rid"`
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

// Biz to SFU

type ToSfuJoinMsg struct {
=======
type OfferMsg struct {
=======
type FromSfuAnswerMsg struct {
>>>>>>> Update SFU node to use ion-sfu.
	RoomInfo
	RTCInfo
}

<<<<<<< HEAD
type AnswerMsg struct {
>>>>>>> Add offer/answer hooks.
	RoomInfo
	RTCInfo
}

<<<<<<< HEAD
type FromSfuJoinMsg struct {
	MediaInfo
	RTCInfo
}

type ToSfuLeaveMsg struct {
	RoomInfo
}

type FromSfuLeaveMsg struct {
	MediaInfo
}

type ToSfuTrickleMsg struct {
	RoomInfo
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type FromSfuTrickleMsg struct {
	RoomInfo
	Candidate webrtc.ICECandidateInit `json:"candidate"`
}

type ToSfuOfferMsg struct {
=======
type ToSfuAnswerMsg struct {
>>>>>>> Update SFU node to use ion-sfu.
	RoomInfo
	RTCInfo
}

<<<<<<< HEAD
type FromSfuOfferMsg struct {
	RoomInfo
	RTCInfo
}

type FromSfuAnswerMsg struct {
	RoomInfo
	RTCInfo
}
=======
// Islb messages
>>>>>>> Latest changes.

type IslbBroadcastMsg struct {
	RoomInfo
	Info json.RawMessage `json:"info"`
}

<<<<<<< HEAD
=======
>>>>>>> Update SFU node to use ion-sfu.
type FromClientBroadcastMsg struct {
	RoomInfo
	Info json.RawMessage `json:"info"`
}

<<<<<<< HEAD
=======
>>>>>>> Handle join with ion-sfu.
=======
>>>>>>> Add offer/answer hooks.
=======
>>>>>>> Update SFU node to use ion-sfu.
type LeaveMsg struct {
	RoomInfo
	Info ClientUserInfo `json:"info"`
=======
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
>>>>>>> Latest changes.
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

type ToIslbFindSfuMsg struct {
	UID UID `json:"uid"`
	RID RID `json:"rid"`
	MID MID `json:"mid"`
}

type FromIslbFindSfuMsg struct {
	RPCID   string
	EventID string
	ID      string
	Name    string
	Service string
}

<<<<<<< HEAD
type StreamAddMsg struct {
	MediaInfo
	Info        ClientUserInfo `json:"info"`
	Tracks      TrackMap       `json:"tracks"`
	Description string         `json:"description,omitempty"`
=======
type ToIslbListMids struct {
	UID UID `json:"uid"`
	RID RID `json:"rid"`
>>>>>>> Latest changes.
}

type FromIslbListMids struct {
	MIDs []MID `json:"mids"`
}
