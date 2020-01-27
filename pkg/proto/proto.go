package proto

import (
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

	// ion to client
	ClientOnJoin         = "peer-join"
	ClientOnLeave        = "peer-leave"
	ClientOnStreamAdd    = "stream-add"
	ClientOnStreamRemove = "stream-remove"

	// ion to islb
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

	IslbID = "islb"
)

func GetUIDFromMID(mid string) string {
	return strings.Split(mid, "#")[0]
}

func GetUserInfoPath(rid, uid string) string {
	return rid + "/user/info/" + uid
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
