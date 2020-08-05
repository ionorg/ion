package proto

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

type ClientUserInfo struct {
	Name string `json:"name"`
}

func (m *ClientUserInfo) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m *ClientUserInfo) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, m)
}

type RoomInfo struct {
	RID RID `json:"rid"`
	UID UID `json:"uid"`
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
// TODO(kevmo314): Consolidate these messages.

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
	Info ClientUserInfo `json:"info"`
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
	MediaInfo
	RTCInfo
}

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
	RoomInfo
	RTCInfo
}

type FromSfuJoinMsg struct {
	MediaInfo
	RTCInfo
}

type ToSfuLeaveMsg struct {
	RoomInfo
}

type FromSfuLeaveMsg struct {
>>>>>>> Update SFU node to use ion-sfu.
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
	RoomInfo
	RTCInfo
}

type FromSfuOfferMsg struct {
	RoomInfo
	RTCInfo
}

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

type ToSfuAnswerMsg struct {
	RoomInfo
	RTCInfo
}

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
}

type PublishMsg struct {
	RoomInfo
	RTCInfo
	Options PublishOptions `json:"options"`
}

type PublishResponseMsg struct {
	MediaInfo
	RTCInfo
	Tracks TrackMap `json:"tracks"`
}

type UnpublishMsg struct {
	MediaInfo
}

type SFUSubscribeMsg struct {
	SubscribeMsg
	Tracks TrackMap `json:"tracks"`
}

type SubscribeMsg struct {
	MediaInfo
	RTCInfo
	Options SubscribeOptions
}

type SubscribeResponseMsg struct {
	MediaInfo
	RTCInfo
}

type UnsubscribeMsg struct {
	MediaInfo
}

type StreamAddMsg struct {
	MediaInfo
	Info        ClientUserInfo `json:"info"`
	Tracks      TrackMap       `json:"tracks"`
	Description string         `json:"description,omitempty"`
}

type StreamRemoveMsg struct {
	MediaInfo
}
