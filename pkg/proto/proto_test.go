package proto

import (
	"fmt"
	"testing"
)

func TestKeyBuildAndParse(t *testing.T) {
	key := MediaInfo{
		DC:  "dc1",
		NID: "sfu1",
		SID: "room1",
		UID: "uid1",
		MID: "mid1",
	}.BuildKey()

	if key != "dc1/sfu1/room1/uid1/media/pub/mid1" {
		t.Error("MediaInfo key not match")
	}
	fmt.Println(key)

	minfo, err := ParseMediaInfo(key)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(minfo)

	key = UserInfo{
		DC:  "dc1",
		SID: "room1",
		UID: "user1",
	}.BuildKey()
	if key != "dc1/room1/user/info/user1" {
		t.Error("UserInfo key not match")
	}
	fmt.Println(key)

	uinfo, _ := ParseUserInfo(key)
	fmt.Println(uinfo)
}

func TestMarshal(t *testing.T) {
	var tracks []TrackInfo
	tracks = append(tracks, TrackInfo{Ssrc: 3694449886, Payload: 96, Type: "audio", ID: "aid"})
	key, value, err := MarshalTrackField("msidxxxxxx", tracks)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("TrackField: key = %s => %s\n", key, value)

	key, value, err = MarshalNodeField(NodeInfo{Name: "sfu001", ID: "uuid-xxxxx-xxxx", Type: "origin"})
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("NodeField: key = %s => %s\n", key, value)
}

func TestUnMarshal(t *testing.T) {
	node, err := UnmarshalNodeField("node/uuid-xxxxx-xxxx", `{"name": "sfu001", "id": "uuid-xxxxx-xxxx", "type": "origin"}`)
	if err == nil {
		fmt.Printf("node => %v\n", node)
	}
	msid, tracks, err := UnmarshalTrackField("track/pion audio", `[{"ssrc": 3694449886, "pt": 111, "type": "audio", "id": "aid"}]`)
	if err != nil {
		t.Errorf("err => %v", err)
	}
	fmt.Printf("msid => %s, tracks => %v\n", msid, tracks)
}
