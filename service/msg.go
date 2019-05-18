package service

import (
	"encoding/json"

	"github.com/pion/sfu/log"
)

const (
	ReqJoin        = "join"
	ReqLeave       = "leave"
	ReqPublish     = "publish"
	ReqOnPublish   = "onPublish"
	ReqSubscribe   = "subscribe"
	ReqOnUnpublish = "onUnpublish"
)

type ReqMsg struct {
	Req    string                 `json:"req"`
	ID     int                    `json:"id"`
	Msg    map[string]interface{} `json:"msg"`
	client string
}

func ReqUnmarshal(data string) ReqMsg {
	r := ReqMsg{}
	err := json.Unmarshal([]byte(data), &r)
	if err != nil {
		log.Errorf("ReqUnmarshal %v", err)
	}
	return r
}

func ReqMarshal(r ReqMsg) string {
	str, err := json.Marshal(r)
	if err != nil {
		log.Errorf("ReqUnmarshal %v", err)
	}
	return string(str)
}

type RespMsg struct {
	Resp   string                 `json:"resp"`
	ID     int                    `json:"id"`
	Msg    map[string]interface{} `json:"msg"`
	client string
}

func RespMarshal(r RespMsg) string {
	str, err := json.Marshal(r)
	if err != nil {
		log.Errorf("RespMarshal %v", err)
	}
	return string(str)
}

func RespUnmarshal(data string) RespMsg {
	r := RespMsg{}
	err := json.Unmarshal([]byte(data), &r)
	if err != nil {
		log.Errorf("RespUnmarshal %v", err)
	}
	return r
}
