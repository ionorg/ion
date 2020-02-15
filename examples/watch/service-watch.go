package main

import (
	"encoding/json"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
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
	sw := node.NewServiceWatcher([]string{"http://127.0.0.1:2379"})
	protoo := nprotoo.NewNatsProtoo("nats://127.0.0.1:4222")
	go sw.WatchServiceNode("sfu", func(service string, nodes []discovery.Node) {
		log.Infof("Service [%s] => %v", service, nodes)
		for _, nd := range nodes {
			req := protoo.NewRequestor(node.GetRPCChannel(nd))
			req.Request("offer", JsonEncode(`{ "sdp": "dummy-sdp"}`),
				func(result map[string]interface{}) {
					log.Infof("offer success: =>  %s", result)
				},
				func(code int, err string) {
					log.Warnf("offer reject: %d => %s", code, err)
				})
		}
	})

	select {}
}
