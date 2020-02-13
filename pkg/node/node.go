package node

import (
	"math/rand"
	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
)

const (
	DefaultEtcdURL = "http://127.0.0.1:2379"
)

// ServiceNode .
type ServiceNode struct {
	reg  *discovery.ServiceRegistry
	np   *nprotoo.NatsProtoo
	name string
	node discovery.Node
}

// NewServiceNode .
func NewServiceNode() *ServiceNode {
	var sn ServiceNode
	etcdURL := DefaultEtcdURL
	sn.reg = discovery.NewServiceRegistry([]string{etcdURL}, "services:/")
	log.Infof("New Service Node: etcd => %s", etcdURL)
	sn.node = discovery.Node{
		Info: make(map[string]string),
	}
	sn.np = nprotoo.NewNatsProtoo(nprotoo.DefaultNatsURL)
	return &sn
}

// GetNatsProtoo .
func (sn *ServiceNode) GetNatsProtoo() *nprotoo.NatsProtoo {
	return sn.np
}

// NewRequestor .
func (sn *ServiceNode) NewRequestor(channel string) *nprotoo.Requestor {
	return sn.np.NewRequestor("rpc-" + channel)
}

// OnRequest .
func (sn *ServiceNode) OnRequest(listener nprotoo.RequestFunc) {
	channel := "rpc-" + sn.node.Info["id"]
	log.Infof("Listen request channel => %s", channel)
	sn.np.OnRequest(channel, listener)
}

// NewBroadcaster .
func (sn *ServiceNode) NewBroadcaster(channel string) *nprotoo.Broadcaster {
	return sn.np.NewBroadcaster("event-" + channel)
}

// OnBroadcast .
func (sn *ServiceNode) OnBroadcast(listener func(data map[string]interface{}, subj string)) {
	channel := "event-" + sn.node.Info["id"]
	log.Infof("Listen broadcast channel => %s", channel)
	sn.np.OnBroadcast(channel, listener)
}

// RegisterNode register a new node.
func (sn *ServiceNode) RegisterNode(serviceName string, name string, ID string) {
	sn.node.Name = randomString(12)
	sn.node.Info["name"] = name
	sn.node.Info["service"] = serviceName
	sn.node.Info["id"] = ID
	err := sn.reg.RegisterServiceNode(serviceName, sn.node)
	if err != nil {
		log.Panicf("%v", err)
	}
}

//WatchServiceNode .
func (sn *ServiceNode) WatchServiceNode(serviceName string, callback discovery.ServiceWatchCallback) {
	for {
		nodes, err := sn.reg.GetServiceNodes(serviceName)
		if err != nil {
			log.Panicf("%v", err)
		}
		callback(serviceName, nodes)
		log.Debugf("Nodes: %v", nodes)
		time.Sleep(2 * time.Second)
	}
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func randomString(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
