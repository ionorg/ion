package biz

import "strings"

func getUIDFromMID(mid string) string {
	return strings.Split(mid, "#")[0]
}

func getUserInfoPath(rid, uid string) string {
	return rid + "/user/info/" + uid
}

func getPubNodePath(rid, uid string) string {
	return rid + "/node/pub/" + uid
}

func getPubMediaPath(rid, mid string) string {
	return rid + "/media/pub/" + mid
}

func getPubMediaPathKey(rid string) string {
	return rid + "/media/pub/*"
}
