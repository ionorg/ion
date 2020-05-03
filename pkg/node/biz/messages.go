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

type PublishOptions struct {
	Codec      string `json:"codec"`
	Resolution string `json:"resolution"`
	Bandwidth  int    `json:"bandwidth"`
	Audio      bool   `json:"audio"`
	Video      bool   `json:"video"`
	Screen     bool   `json:"screen"`
}

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
	Jsep    webrtc.SessionDescription `json:"jsep"`
	Options PublishOptions            `json:"options"`
}

type UnpublishMsg struct {
	RoomInfo
	MediaInfo
	Options PublishOptions `json:"options"`
}

type SubscribeMsg struct {
	MediaInfo
	Jsep webrtc.SessionDescription `json:"jsep"`
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
