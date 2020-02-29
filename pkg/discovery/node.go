package discovery

import (
	"math/rand"
	"time"

	"github.com/pion/ion/pkg/log"
	"go.etcd.io/etcd/clientv3"
)

// ServiceNode .
type ServiceNode struct {
	reg  *ServiceRegistry
	name string
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

// RegisterNode register a new node.
func (sn *ServiceNode) RegisterNode(serviceName string, name string, ID string) {
	sn.node.ID = randomString(12)
	sn.node.Info["name"] = name
	sn.node.Info["service"] = serviceName
	sn.node.Info["id"] = serviceName + "-" + sn.node.ID
	err := sn.reg.RegisterServiceNode(serviceName, sn.node)
	if err != nil {
		log.Panicf("%v", err)
	}
}

// ServiceWatcher .
type ServiceWatcher struct {
	reg      *ServiceRegistry
	nodesMap map[string]map[string]Node
}

// NewServiceWatcher .
func NewServiceWatcher(endpoints []string, dc string) *ServiceWatcher {
	sw := &ServiceWatcher{
		nodesMap: make(map[string]map[string]Node),
		reg:      NewServiceRegistry(endpoints, "/"+dc+"/node/"),
	}
	log.Infof("New Service Watcher: etcd => %v", endpoints)
	return sw
}

func (sw *ServiceWatcher) GetNodes(service string) (map[string]Node, bool) {
	nodes, found := sw.nodesMap[service]
	return nodes, found
}

func (sw *ServiceWatcher) GetNodesByID(service string, ID string) (*Node, bool) {
	nodes, found := sw.nodesMap[service]
	if found {
		for id, node := range nodes {
			if id == ID {
				return &node, true
			}
			return nil, false
		}
	}
	return nil, false
}

//WatchServiceNode .
func (sw *ServiceWatcher) WatchServiceNode(serviceName string, callback ServiceWatchCallback) {
	log.Infof("Start service watcher => [%s].", serviceName)

	for {
		nodes, err := sw.reg.GetServiceNodes(serviceName)
		if err != nil {
			log.Panicf("%v", err)
		}
		log.Debugf("Nodes: => %v", nodes)

		for _, node := range nodes {
			id := node.ID
			service := node.Info["service"]
			if _, found := sw.GetNodesByID(service, node.ID); !found {
				log.Infof("New %s node UP => [%s].", service, node.ID)
				callback(service, UP, node)
				Watch(node.ID, func(ch clientv3.WatchChan) {
					log.Infof("Watch %s node => [%s].", service, node.ID)
					go func() {
						for {
							msg := <-ch
							for _, ev := range msg.Events {
								log.Infof("%s %q:%q", ev.Type, ev.Kv.Key, ev.Kv.Value)
								if ev.Type == clientv3.EventTypeDelete {
									log.Infof("Node %s Down => [%s].", service, node.ID)
									delete(sw.nodesMap[service], id)
									callback(service, DOWN, node)
								}
							}
						}
					}()
				}, true)
				if _, found := sw.nodesMap[service]; !found {
					sw.nodesMap[service] = make(map[string]Node)
				}
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
