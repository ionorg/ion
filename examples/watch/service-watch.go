package main

import (
	"encoding/json"

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
	sn := node.NewServiceNode()
	go sn.WatchServiceNode("SFU", func(service string, nodes []discovery.Node) {
		log.Infof("Service [%s] => %v", service, nodes)
		for _, item := range nodes {
			req := sn.NewRequestor(item.Info["id"])
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
