package biz

import (
	"encoding/json"

	"github.com/pion/webrtc/v2"
)

type UserInfo struct {
	Name string `json:"name"`
}

type RoomInfo struct {
	Rid string `json:"rid"`
	Uid string `json:"uid"`
}

type MediaInfo struct {
	Mid string `json:"mid"`
}

type RTCInfo struct {
	Jsep webrtc.SessionDescription `json:"jsep"`
}

type TrackInfo struct {
	Type  string `json:"type"`
	Codec string `json:"codec"`
	FMTP  string `json:"fmtp"`
	ID    string `json:"id"`
	PT    int    `json:"pt"`
	SSRC  int64  `json:"ssrc"`
}

type PublishOptions struct {
	Codec      string `json:"codec"`
	Resolution string `json:"resolution"`
	Bandwidth  int    `json:"bandwidth"`
	Audio      bool   `json:"audio"`
	Video      bool   `json:"video"`
	Screen     bool   `json:"screen"`
}

type TrackMap map[string][]TrackInfo

/// Messages ///

type JoinMsg struct {
	RoomInfo
	Info UserInfo `json:"info"`
}

type LoginMsg struct {
}

type LeaveMsg struct {
	RoomInfo
	Info UserInfo `json:"info"`
}

type CloseMsg struct {
	LeaveMsg
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
	RoomInfo
	MediaInfo
}

type SubscribeMsg struct {
	RoomInfo
	MediaInfo
	RTCInfo
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
	RoomInfo
	MediaInfo
	Info    json.RawMessage `json:"info"`
	Trickle json.RawMessage `json:"trickle"`
}

type StreamAddMsg struct {
	RoomInfo
	MediaInfo
	Info   UserInfo `json:"info"`
	Tracks TrackMap `json:"tracks"`
}

type StreamRemoveMsg struct {
	RoomInfo
	MediaInfo
}
