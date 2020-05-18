package proto

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	// client to ion
	ClientLogin       = "login"
	ClientJoin        = "join"
	ClientLeave       = "leave"
	ClientPublish     = "publish"
	ClientUnPublish   = "unpublish"
	ClientSubscribe   = "subscribe"
	ClientUnSubscribe = "unsubscribe"
	ClientClose       = "close"
	ClientBroadcast   = "broadcast"
	ClientTrickleICE  = "trickle"

	// ion to client
	ClientOnJoin         = "peer-join"
	ClientOnLeave        = "peer-leave"
	ClientOnStreamAdd    = "stream-add"
	ClientOnStreamRemove = "stream-remove"

	// ion to islb
	IslbFindService  = "findService"
	IslbGetPubs      = "getPubs"
	IslbGetMediaInfo = "getMediaInfo"
	IslbRelay        = "relay"
	IslbUnrelay      = "unRelay"

	IslbKeepAlive      = "keepAlive"
	IslbClientOnJoin   = ClientOnJoin
	IslbClientOnLeave  = ClientOnLeave
	IslbOnStreamAdd    = ClientOnStreamAdd
	IslbOnStreamRemove = ClientOnStreamRemove
	IslbOnBroadcast    = ClientBroadcast

	SFUTrickleICE   = ClientTrickleICE
	SFUStreamRemove = ClientOnStreamRemove

	IslbID = "islb"
)

/*
media
dc/${nid}/${rid}/${uid}/media/pub/${mid}

node1 origin
node2 shadow
msid  [{ssrc: 1234, pt: 111, type:audio}]
msid  [{ssrc: 5678, pt: 96, type:video}]
*/

func BuildMediaInfoKey(dc string, nid string, rid string, uid string, mid string) string {
	strs := []string{dc, nid, rid, uid, "media", "pub", mid}
	return strings.Join(strs, "/")
}

type MediaInfo struct {
	DC  string //Data Center ID
	NID string //Node ID
	RID string //Room ID
	UID string //User ID
	MID string //Media ID
}

// ParseMediaInfo dc1/sfu-tU2GInE5Lfuc/7485294b-9815-4888-83a5-631e77445b67/room1/media/pub/7e97c1e8-c80a-4c69-81b0-27efc83e6120
func ParseMediaInfo(key string) (*MediaInfo, error) {
	var info MediaInfo
	arr := strings.Split(key, "/")
	if len(arr) != 7 {
		return nil, fmt.Errorf("Can‘t parse mediainfo; [%s]", key)
	}
	info.DC = arr[0]
	info.NID = arr[1]
	info.RID = arr[2]
	info.UID = arr[3]
	info.MID = arr[6]
	return &info, nil
}

/*
user
/dc/room1/user/info/${uid}
info {name: "Guest"}
*/

func BuildUserInfoKey(dc string, rid string, uid string) string {
	strs := []string{dc, rid, "user", "info", uid}
	return strings.Join(strs, "/")
}

type UserInfo struct {
	DC  string
	RID string
	UID string
}

func ParseUserInfo(key string) (*UserInfo, error) {
	var info UserInfo
	arr := strings.Split(key, "/")
	if len(arr) != 5 {
		return nil, fmt.Errorf("Can‘t parse userinfo; [%s]", key)
	}
	info.DC = arr[0]
	info.RID = arr[1]
	info.UID = arr[4]
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

func GetRIDMIDUIDFromMediaKey(key string) (string, string, string) {
	//room1/media/pub/74baff6e-b8c9-4868-9055-b35d50b73ed6#LUMGUQ/11111
	strs := strings.Split(key, "/")
	if len(strs) < 2 {
		return "", "", ""
	}
	strss := strings.Split(strs[3], "#")
	if len(strss) < 2 {
		return "", "", ""
	}
	return strs[0], strs[3], strss[0]
}
