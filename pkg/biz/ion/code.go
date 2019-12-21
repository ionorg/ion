package biz

type code int

const (
	// OK is returned on success.
	codeOK int = -iota
	codeUnknownErr
	codeJsepErr
	codeSDPErr
	codeRoomErr
	codePubIDErr
	codeMIDErr
	codeAddrErr
	codeUIDErr
	codePublishErr
)

var codeErr = map[int]string{
	codeOK:         "OK",
	codeUnknownErr: "unknown error",
	codeJsepErr:    "jsep not found",
	codeSDPErr:     "sdp not found",
	codeRoomErr:    "room not found",
	codePubIDErr:   "pub id not found",
	codeMIDErr:     "media id not found",
	codeAddrErr:    "addr not found",
	codeUIDErr:     "uid not found",
	codePublishErr: "publish failed",
}

func codeStr(code int) string {
	return codeErr[code]
}

var emptyMap = map[string]interface{}{}
