package main

import (
	"encoding/json"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/node"
)

func JsonEncode(str string) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		panic(err)
	}
	return data
}

func main() {
	node.Init()
	node := node.NewServiceNode()

	node.RegisterNode("SFU", "node-name", "nats-channel-bbbbbb-cccccc-dddddd")

	node.OnRequest(func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		method := request["method"].(string)
		data := request["data"].(map[string]interface{})
		log.Infof("method => %s, data => %v", method, data)

		//accept(JsonEncode(`{"answer": "dummy-sdp2"}`))
		reject(404, "Not found")
	})
	node.OnBroadcast(func(data map[string]interface{}, subj string) {
		log.Infof("Got Broadcast subj => %s, data => %v", subj, data)
	})

	//broadcaster := node.NewBroadcaster("uuid-bbbbbb-cccccc-dddddd")
	//broadcaster.Say("foo", JsonEncode(`{"hello": "world"}`))
	select {}
}
