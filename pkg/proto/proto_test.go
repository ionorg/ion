package proto

import (
	"fmt"
	"testing"
)

func TestKeyBuildAndParse(t *testing.T) {
	key := MediaInfo{
		DC:  "dc1",
		NID: "sfu1",
		RID: "room1",
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
		RID: "room1",
		UID: "user1",
	}.BuildKey()
	if key != "dc1/room1/user/info/user1" {
		t.Error("UserInfo key not match")
	}
	fmt.Println(key)

	uinfo, _ := ParseUserInfo(key)
	fmt.Println(uinfo)
}
