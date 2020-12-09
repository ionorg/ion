package db

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pion/ion/pkg/proto"
)

var (
	db     *Redis
	dc     = "dc1"
	node   = "sfu1"
	room   = proto.SID("room1")
	uid    = proto.UID("uuid-xxxxx-xxxxx-xxxxx-xxxxx")
	mid    = proto.MID("mid-xxxxx-xxxxx-xxxxx-xxxxx")
	msid0  = "pion audio"
	msid1  = "pion video"
	track0 = proto.TrackInfo{Ssrc: 3694449886, Payload: 111, Type: "audio", ID: "aid0"}
	track1 = proto.TrackInfo{Ssrc: 8888888888, Payload: 96, Type: "video", ID: "vid1"}
	track2 = proto.TrackInfo{Ssrc: 6666666666, Payload: 117, Type: "video", ID: "vid2"}
	node0  = proto.NodeInfo{Name: "node-name-01", ID: "node-id-01", Type: "origin"}
	node1  = proto.NodeInfo{Name: "node-name-02", ID: "node-id-02", Type: "shadow"}

	uikey = "info"
	uinfo = `{"name": "Guest"}`

	mkey = proto.MediaInfo{
		DC:  dc,
		NID: node,
		SID: room,
		UID: uid,
		MID: mid,
	}.BuildKey()
	ukey = proto.UserInfo{
		DC:  dc,
		SID: room,
		UID: uid,
	}.BuildKey()
)

func init() {
	cfg := Config{
		Addrs: []string{":6379"},
		Pwd:   "",
		DB:    0,
	}
	db = NewRedis(cfg)
}

func TestRedisStorage(t *testing.T) {
	tracks := []proto.TrackInfo{track0}
	field, value, err := proto.MarshalTrackField(msid0, tracks)
	if err != nil {
		t.Error(err)
	}
	t.Logf("HSet Track %s, %s => %s\n", mkey, field, value)
	err = db.HSet(mkey, field, value)
	if err != nil {
		t.Error(err)
	}

	tracks = []proto.TrackInfo{track1, track2}
	field, value, err = proto.MarshalTrackField(msid1, tracks)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("HSet Track %s, %s => %s\n", mkey, field, value)
	err = db.HSet(mkey, field, value)
	if err != nil {
		t.Error(err)
	}

	field, value, err = proto.MarshalNodeField(node0)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("HSet Node %s, %s => %s\n", mkey, field, value)
	err = db.HSet(mkey, field, value)
	if err != nil {
		t.Error(err)
	}

	field, value, err = proto.MarshalNodeField(node1)
	if err != nil {
		t.Error(err)
	}
	fmt.Printf("HSet Node %s, %s => %s\n", mkey, field, value)
	err = db.HSet(mkey, field, value)
	if err != nil {
		t.Error(err)
	}

	fmt.Printf("HSet UserInfo %s, %s => %s\n", ukey, uikey, uinfo)
	err = db.HSet(ukey, uikey, uinfo)
	if err != nil {
		t.Error(err)
	}
}

func TestRedisRead(t *testing.T) {

	fields := db.HGetAll(mkey)

	for key, value := range fields {
		if strings.HasPrefix(key, "node/") {
			node, err := proto.UnmarshalNodeField(key, value)
			if err != nil {
				t.Error(err)
			}
			fmt.Printf("node => %v\n", node)
			if node.ID == "node-id-01" && *node != node0 {
				t.Error("node0 not equal")
			}

			if node.ID == "node-id-02" && *node != node1 {
				t.Error("node1 not equal")
			}
		}
		if strings.HasPrefix(key, "track/") {
			msid, tracks, err := proto.UnmarshalTrackField(key, value)
			if err != nil {
				t.Error(err)
			}
			fmt.Printf("msid => %s, tracks => %v\n", msid, tracks)

			if msid == msid0 && len(*tracks) != 1 {
				t.Error("track0 not equal")
			}

			if msid == msid1 && len(*tracks) != 2 {
				t.Error("track1 not equal")
			}
		}
	}

	fields = db.HGetAll(ukey)
	for key, value := range fields {
		fmt.Printf("key => %s, value => %v\n", key, value)
		if key != uikey && value != uinfo {
			t.Errorf("Failed %s => %s", key, value)
		}
	}
}
