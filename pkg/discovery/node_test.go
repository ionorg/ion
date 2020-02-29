package discovery

import (
	"encoding/json"
	"sync"
	"testing"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	log "github.com/pion/ion/pkg/log"
)

const (
	EtcdAddr = "http://127.0.0.1:2379"
)

var (
	wg    *sync.WaitGroup
	nodes []Node
)

func init() {
	log.Init("info")
	wg = new(sync.WaitGroup)
}

func JsonEncode(str string) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		panic(err)
	}
	return data
}

func ServiceNodeRegistry() {
	serviceNode := NewServiceNode([]string{EtcdAddr}, "dc1")
	serviceNode.RegisterNode("sfu", "node-name", "nats-channel-test")
	protoo := nprotoo.NewNatsProtoo(nprotoo.DefaultNatsURL)
	wg.Add(1)
	protoo.OnRequest(serviceNode.GetRPCChannel(), func(request map[string]interface{}, accept nprotoo.AcceptFunc, reject nprotoo.RejectFunc) {
		method := request["method"].(string)
		data := request["data"].(map[string]interface{})
		log.Infof("method => %s, data => %v", method, data)
		reject(404, "Not found")
		wg.Done()
	})

	protoo.OnBroadcast(serviceNode.GetEventChannel(), func(data map[string]interface{}, subj string) {
		log.Infof("Got Broadcast subj => %s, data => %v", subj, data)
		wg.Done()
	})

	wg.Add(1)
	broadcaster := protoo.NewBroadcaster(GetEventChannel(serviceNode.NodeInfo()))
	broadcaster.Say("foo", JsonEncode(`{"hello": "world"}`))
}

func ServiceNodeWatch() {
	serviceWatcher := NewServiceWatcher([]string{EtcdAddr}, "dc1")
	protoo := nprotoo.NewNatsProtoo(nprotoo.DefaultNatsURL)
	go serviceWatcher.WatchServiceNode("sfu", func(service string, state NodeStateType, node Node) {
		if state == UP {
			log.Infof("Service UP [%s] => %v", service, node)
			req := protoo.NewRequestor(GetRPCChannel(node))
			wg.Add(1)
			req.Request("offer", JsonEncode(`{ "sdp": "dummy-sdp"}`),
				func(result map[string]interface{}) {
					log.Infof("offer success: =>  %s", result)
				},
				func(code int, err string) {
					log.Warnf("offer reject: %d => %s", code, err)
					wg.Done()
				})
		} else if state == DOWN {
			log.Infof("Service DOWN [%s] => %v", service, node)

		}
	})
}

func TestServiceNode(t *testing.T) {
	ServiceNodeWatch()
	ServiceNodeRegistry()
	wg.Wait()
}
