package proto

import (
	"encoding/json"

	"github.com/pion/webrtc/v2"
)

type ClientUserInfo struct {
	Name string `json:"name"`
}

type RoomInfo struct {
	Rid string `json:"rid"`
	Uid string `json:"uid"`
}

type RTCInfo struct {
	Jsep webrtc.SessionDescription `json:"jsep"`
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
	Info ClientUserInfo `json:"info"`
}

type LoginMsg struct {
}

type LeaveMsg struct {
	RoomInfo
	Info ClientUserInfo `json:"info"`
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
	MediaInfo
}

type SubscribeMsg struct {
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
	MediaInfo
	Info    json.RawMessage `json:"info"`
	Trickle json.RawMessage `json:"trickle"`
}

type StreamAddMsg struct {
	MediaInfo
	Info   ClientUserInfo `json:"info"`
	Tracks TrackMap       `json:"tracks"`
}

type StreamRemoveMsg struct {
	MediaInfo
}
