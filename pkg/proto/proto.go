package proto

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// client to ion
	ClientJoin        = "join"
	ClientLeave       = "leave"
	ClientPublish     = "publish"
	ClientUnPublish   = "unpublish"
	ClientSubscribe   = "subscribe"
	ClientUnSubscribe = "unsubscribe"
	ClientBroadcast   = "broadcast"
	ClientTrickleICE  = "trickle"
	ClientOffer       = "offer"
	ClientAnswer      = "answer"

	// ion to client
	ClientOnJoin         = "peer-join"
	ClientOnLeave        = "peer-leave"
	ClientOnStreamAdd    = "stream-add"
	ClientOnStreamRemove = "stream-remove"
	ClientOnOffer        = "offer"
	ClientOnAnswer       = "answer"

	// ion to islb
	IslbFindNode = "find-node"
	IslbRelay    = "relay"
	IslbUnrelay  = "unRelay"

	IslbKeepAlive = "keepAlive"
	IslbPeerJoin  = ClientOnJoin
	IslbPeerLeave = ClientOnLeave
	IslbListMids  = "list-mids"
	IslbStreamAdd = ClientOnStreamAdd
	IslbBroadcast = ClientBroadcast

	// SFU Endpoints
	SfuTrickleICE    = ClientTrickleICE
	SfuClientJoin    = ClientJoin
	SfuClientOffer   = ClientOnOffer
	SfuClientAnswer  = ClientOnAnswer
	SfuClientTrickle = ClientTrickleICE
	SfuClientLeave   = ClientLeave

	IslbID = "islb"
)

type MID string
type SID string
type RID string
type UID string

/*
media
dc/${nid}/${rid}/${uid}/media/pub/${mid}

node1 origin
node2 shadow
msid  [{ssrc: 1234, pt: 111, type:audio}]
msid  [{ssrc: 5678, pt: 96, type:video}]
*/
type MediaInfo struct {
	DC  string `json:"dc,omitempty"`  //Data Center ID
	NID string `json:"nid,omitempty"` //Node ID
	RID RID    `json:"rid,omitempty"` //Room ID
	UID UID    `json:"uid,omitempty"` //User ID
	MID MID    `json:"mid,omitempty"` //Media ID
}

func (m MediaInfo) BuildKey() string {
	if m.DC == "" {
		m.DC = "*"
	}
	if m.NID == "" {
		m.NID = "*"
	}
	if m.RID == "" {
		m.RID = "*"
	}
	if m.UID == "" {
		m.UID = "*"
	}
	if m.MID == "" {
		m.MID = "*"
	}
	strs := []string{m.DC, m.NID, string(m.RID), string(m.UID), "media", "pub", string(m.MID)}
	return strings.Join(strs, "/")
}

// Parse dc1/sfu-tU2GInE5Lfuc/7485294b-9815-4888-83a5-631e77445b67/room1/media/pub/7e97c1e8-c80a-4c69-81b0-27efc83e6120
func ParseMediaInfo(key string) (*MediaInfo, error) {
	var info MediaInfo
	arr := strings.Split(key, "/")
	if len(arr) != 7 {
		return nil, fmt.Errorf("Can‘t parse mediainfo; [%s]", key)
	}
	info.DC = arr[0]
	info.NID = arr[1]
	info.RID = RID(arr[2])
	info.UID = UID(arr[3])
	info.MID = MID(arr[6])
	return &info, nil
}

/*
user
/dc/room1/user/info/${uid}
info {name: "Guest"}
*/

type UserInfo struct {
	DC  string
	RID RID
	UID UID
}

func (u UserInfo) BuildKey() string {
	uid := string(u.UID)
	if uid == "" {
		uid = "*"
	}
	strs := []string{u.DC, string(u.RID), "user", "info", uid}
	return strings.Join(strs, "/")
}

func ParseUserInfo(key string) (*UserInfo, error) {
	var info UserInfo
	arr := strings.Split(key, "/")
	if len(arr) != 5 {
		return nil, fmt.Errorf("Can‘t parse userinfo; [%s]", key)
	}
	info.DC = arr[0]
	info.RID = RID(arr[1])
	info.UID = UID(arr[4])
	return &info, nil
}

type NodeInfo struct {
	Name string `json:"name"`
	ID   string `json:"id"`
	Type string `json:"type"` // origin | shadow
}

func MarshalNodeField(node NodeInfo) (string, string, error) {
	value, err := json.Marshal(node)
	if err != nil {
		return "node/" + node.ID, "", fmt.Errorf("Marshal: %v", err)
	}
	return "node/" + node.ID, string(value), nil
}

func UnmarshalNodeField(key string, value string) (*NodeInfo, error) {
	var node NodeInfo
	if err := json.Unmarshal([]byte(value), &node); err != nil {
		return nil, fmt.Errorf("Unmarshal: %v", err)
	}
	return &node, nil
}

type TrackInfo struct {
	ID      string `json:"id"`
	Ssrc    int    `json:"ssrc"`
	Payload int    `json:"pt"`
	Type    string `json:"type"`
	Codec   string `json:"codec"`
	Fmtp    string `json:"fmtp"`
}

func MarshalTrackField(id string, infos []TrackInfo) (string, string, error) {
	str, err := json.Marshal(infos)
	if err != nil {
		return "track/" + id, "", fmt.Errorf("Marshal: %v", err)
	}
	return "track/" + id, string(str), nil
}

func UnmarshalTrackField(key string, value string) (string, *[]TrackInfo, error) {
	var tracks []TrackInfo
	if err := json.Unmarshal([]byte(value), &tracks); err != nil {
		return "", nil, fmt.Errorf("Unmarshal: %v", err)
	}
	if !strings.Contains(key, "track/") {
		return "", nil, fmt.Errorf("Invalid track failed => %s", key)
	}
	msid := strings.Split(key, "/")[1]
	return msid, &tracks, nil
}

func GetPubNodePath(rid, uid string) string {
	return rid + "/node/pub/" + uid
}

func GetPubMediaPath(rid, mid string, ssrc uint32) string {
	if ssrc != 0 {
		return rid + "/media/pub/" + mid + fmt.Sprintf("/%d", ssrc)
	}
	return rid + "/media/pub/" + mid
}

func GetPubMediaPathKey(rid string) string {
	return rid + "/media/pub/"
}
