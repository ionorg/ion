package proto

import (
	"encoding/json"

	"github.com/pion/webrtc/v2"
)

type Authenticatable interface {
	Room() RoomInfo
	Token() string
}

type ClientUserInfo struct {
	Name string `json:"name"`
}

func (m *ClientUserInfo) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func (m *ClientUserInfo) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, m)
}

type RoomToken struct {
	Token string `json:"token,omitempty"`
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

/// Messages ///

type JoinMsg struct {
	RoomInfo
	RoomToken
	Info ClientUserInfo `json:"info"`
}

func (j *JoinMsg) Token() string {
	return j.RoomToken.Token
}
func (j *JoinMsg) Room() RoomInfo {
	return j.RoomInfo
}

type LeaveMsg struct {
	RoomInfo
	Info ClientUserInfo `json:"info"`
}

type PublishMsg struct {
	RoomInfo
	RoomToken
	RTCInfo
	Options PublishOptions `json:"options"`
}

func (p *PublishMsg) Token() string {
	return p.RoomToken.Token
}
func (p *PublishMsg) Room() RoomInfo {
	return p.RoomInfo
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

type BroadcastMsg struct {
	RoomInfo
	Info json.RawMessage `json:"info"`
}

type TrickleMsg struct {
	MediaInfo
	Info    json.RawMessage `json:"info"`
	Trickle json.RawMessage `json:"trickle"`
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
