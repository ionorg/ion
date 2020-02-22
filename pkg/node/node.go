package node

import (
	"math/rand"
	"time"

	"github.com/pion/ion/pkg/discovery"
	"github.com/pion/ion/pkg/log"
)

// ServiceNode .
type ServiceNode struct {
	reg  *discovery.ServiceRegistry
	name string
	node discovery.Node
}

// NewServiceNode .
func NewServiceNode(endpoints []string) *ServiceNode {
	var sn ServiceNode
	sn.reg = discovery.NewServiceRegistry(endpoints, "services:/")
	log.Infof("New Service Node Registry: etcd => %v", endpoints)
	sn.node = discovery.Node{
		Info: make(map[string]string),
	}
	return &sn
}

// NodeInfo .
func (sn *ServiceNode) NodeInfo() discovery.Node {
	return sn.node
}

// GetEventChannel .
func (sn *ServiceNode) GetEventChannel() string {
	return "event-" + sn.node.Info["id"]
}

// GetRPCChannel .
func (sn *ServiceNode) GetRPCChannel() string {
	return "rpc-" + sn.node.Info["id"]
}

// RegisterNode register a new node.
func (sn *ServiceNode) RegisterNode(serviceName string, name string, ID string) {
	sn.node.Name = randomString(12)
	sn.node.Info["name"] = name
	sn.node.Info["service"] = serviceName
	sn.node.Info["id"] = ID + "-" + sn.node.Name
	err := sn.reg.RegisterServiceNode(serviceName, sn.node)
	if err != nil {
		log.Panicf("%v", err)
	}
}

// ServiceWatcher .
type ServiceWatcher struct {
	reg *discovery.ServiceRegistry
}

// NewServiceWatcher .
func NewServiceWatcher(endpoints []string) *ServiceWatcher {
	var sw ServiceWatcher
	sw.reg = discovery.NewServiceRegistry(endpoints, "services:/")
	log.Infof("New Service Watcher: etcd => %v", endpoints)
	return &sw
}

//WatchServiceNode .
func (sw *ServiceWatcher) WatchServiceNode(serviceName string, callback discovery.ServiceWatchCallback) {
	log.Infof("Start service watcher => [%s].", serviceName)
	for {
		nodes, err := sw.reg.GetServiceNodes(serviceName)
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

// GetEventChannel .
func GetEventChannel(node discovery.Node) string {
	return "event-" + node.Info["id"]
}

// GetRPCChannel .
func GetRPCChannel(node discovery.Node) string {
	return "rpc-" + node.Info["id"]
}
