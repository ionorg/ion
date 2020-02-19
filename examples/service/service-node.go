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
	sn := node.NewServiceNode([]string{"http://127.0.0.1:2379"})
	sn.RegisterNode("sfu", "node-name", "nats-channel-test")
	protoo := nprotoo.NewNatsProtoo("nats://127.0.0.1:4222")
	protoo.OnRequest(sn.GetRPCChannel(), func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		method := request["method"].(string)
		data := request["data"].(map[string]interface{})
		log.Infof("method => %s, data => %v", method, data)

		//accept(JsonEncode(`{"answer": "dummy-sdp2"}`))
		reject(404, "Not found")
	})

	protoo.OnBroadcast(sn.GetEventChannel(), func(data map[string]interface{}, subj string) {
		log.Infof("Got Broadcast subj => %s, data => %v", subj, data)
	})

	broadcaster := protoo.NewBroadcaster(node.GetEventChannel(sn.NodeInfo()))
	broadcaster.Say("foo", JsonEncode(`{"hello": "world"}`))
	select {}
}
