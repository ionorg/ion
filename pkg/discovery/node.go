package discovery

import (
	"math/rand"
	"time"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/util"
	"go.etcd.io/etcd/clientv3"
)

// ServiceNode .
type ServiceNode struct {
	reg  *ServiceRegistry
	node Node
}

type NodeStateType int32

const (
	UP   NodeStateType = 0
	DOWN NodeStateType = 1
)

// ServiceWatchCallback .
type ServiceWatchCallback func(service string, state NodeStateType, nodes Node)

// NewServiceNode .
func NewServiceNode(endpoints []string, dc string) *ServiceNode {
	var sn ServiceNode
	sn.reg = NewServiceRegistry(endpoints, "/"+dc+"/node/")
	log.Infof("New Service Node Registry: etcd => %v", endpoints)
	sn.node = Node{
		Info: make(map[string]string),
	}
	return &sn
}

// NodeInfo .
func (sn *ServiceNode) NodeInfo() Node {
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

// GetGRPCAddress .
func (sn *ServiceNode) GetGRPCAddress() string {
	return sn.node.Info["grpc"]
}

// RegisterNode register a new node.
func (sn *ServiceNode) RegisterNode(serviceName, name, ID, grpcPort string) {
	sn.node.ID = randomString(12)
	sn.node.Info["name"] = name
	sn.node.Info["service"] = serviceName
	sn.node.Info["id"] = serviceName + "-" + sn.node.ID
	sn.node.Info["ip"] = util.GetIntefaceIP()
	if grpcPort != "" {
		sn.node.Info["grpc"] = sn.node.Info["ip"] + grpcPort
	}
	err := sn.reg.RegisterServiceNode(serviceName, sn.node)
	if err != nil {
		log.Panicf("%v", err)
	}
}

// ServiceWatcher .
type ServiceWatcher struct {
	reg      *ServiceRegistry
	nodesMap map[string]map[string]Node
	callback ServiceWatchCallback
}

// NewServiceWatcher .
func NewServiceWatcher(endpoints []string, dc string) *ServiceWatcher {
	sw := &ServiceWatcher{
		nodesMap: make(map[string]map[string]Node),
		reg:      NewServiceRegistry(endpoints, "/"+dc+"/node/"),
		callback: nil,
	}
	log.Infof("New Service Watcher: etcd => %v", endpoints)
	return sw
}

func (sw *ServiceWatcher) GetNodes(service string) (map[string]Node, bool) {
	nodes, found := sw.nodesMap[service]
	return nodes, found
}

func (sw *ServiceWatcher) GetNodesByID(ID string) (*Node, bool) {
	for _, nodes := range sw.nodesMap {
		for id, node := range nodes {
			if id == ID {
				return &node, true
			}
		}
	}
	return nil, false
}

func (sw *ServiceWatcher) DeleteNodesByID(ID string) bool {
	for service, nodes := range sw.nodesMap {
		for id := range nodes {
			if id == ID {
				delete(sw.nodesMap[service], id)
				return true
			}
		}
	}
	return false
}

func (sw *ServiceWatcher) WatchNode(ch clientv3.WatchChan) {
	go func() {
		for {
			msg := <-ch
			for _, ev := range msg.Events {
				log.Infof("%s %q:%q", ev.Type, ev.Kv.Key, ev.Kv.Value)
				if ev.Type == clientv3.EventTypeDelete {
					nodeID := string(ev.Kv.Key)
					log.Infof("Node [%s] Down", nodeID)
					n, found := sw.GetNodesByID(nodeID)
					if found {
						service := n.Info["service"]
						if sw.callback != nil {
							sw.callback(service, DOWN, *n)
						}
						sw.DeleteNodesByID(nodeID)
					}
				}
			}
		}
	}()
}

//WatchServiceNode .
func (sw *ServiceWatcher) WatchServiceNode(serviceName string, callback ServiceWatchCallback) {
	log.Infof("Start service watcher => [%s].", serviceName)
	sw.callback = callback
	for {
		nodes, err := sw.reg.GetServiceNodes(serviceName)
		if err != nil {
			log.Warnf("sw.reg.GetServiceNodes err=%v", err)
			continue
		}
		log.Debugf("Nodes: => %v", nodes)

		for _, node := range nodes {
			id := node.ID
			service := node.Info["service"]

			if _, found := sw.nodesMap[service]; !found {
				sw.nodesMap[service] = make(map[string]Node)
			}

			if _, found := sw.GetNodesByID(node.ID); !found {
				log.Infof("New %s node UP => [%s].", service, node.ID)
				callback(service, UP, node)

				log.Infof("Start watch for [%s] node => [%s].", service, node.ID)
				Watch(node.ID, sw.WatchNode, true)
				sw.nodesMap[service][id] = node
			}
		}
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
func GetEventChannel(node Node) string {
	return "event-" + node.Info["id"]
}

// GetRPCChannel .
func GetRPCChannel(node Node) string {
	return "rpc-" + node.Info["id"]
}

// GetGRPCAddress for a node
func GetGRPCAddress(node Node) string {
	return node.Info["grpc"]
}
