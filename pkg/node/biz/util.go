package biz

import (
	"fmt"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/util"
)

func getMID(uid string) string {
	return fmt.Sprintf("%s#%s", uid, util.RandStr(6))
}

func getKeepAliveID(mid string, ssrc uint32) string {
	return fmt.Sprintf("%s#%d", mid, ssrc)
}

func verifyData(msg map[string]interface{}, args ...interface{}) (bool, *nprotoo.Error) {
	for i := 0; i < len(args); i++ {
		failed, err := invalid(msg, args[i].(string))
		if failed {
			return !failed, err
		}
	}
	return true, nil
}

func invalid(msg map[string]interface{}, key string) (bool, *nprotoo.Error) {
	val := util.Val(msg, key)
	if val == "" {
		switch key {
		case "rid":
			return true, util.NewNpError(codeRoomErr, codeStr(codeRoomErr))
		case "jsep":
			return true, util.NewNpError(codeJsepErr, codeStr(codeJsepErr))
		case "sdp":
			return true, util.NewNpError(codeSDPErr, codeStr(codeSDPErr))
		case "mid":
			return true, util.NewNpError(codeMIDErr, codeStr(codeMIDErr))
		}
	}
	return false, nil
}
