package islb

import (
	"sync"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	"github.com/cloudwebrtc/nats-discovery/pkg/registry"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/db"
	"github.com/pion/ion/pkg/proto"
)

type Registry struct {
	dc    string
	redis *db.Redis
	reg   *registry.Registry
	mutex sync.Mutex
	nodes map[string]discovery.Node
}

func NewRegistry(dc string, nc *nats.Conn, redis *db.Redis) (*Registry, error) {

	reg, err := registry.NewRegistry(nc, discovery.DefaultExpire)
	if err != nil {
		log.Errorf("registry.NewRegistry: error => %v", err)
		return nil, err
	}

	r := &Registry{
		dc:    dc,
		reg:   reg,
		redis: redis,
		nodes: make(map[string]discovery.Node),
	}

	err = reg.Listen(r.handleNodeAction, r.handleGetNodes)

	if err != nil {
		log.Errorf("registry.Listen: error => %v", err)
		r.Close()
		return nil, err
	}

	return r, nil
}

func (r *Registry) Close() {
	r.reg.Close()
}

// handleNodeAction handle all Node from service discovery.
// This callback can observe all nodes in the ion cluster,
// TODO: Upload all node information to redis DB so that info
// can be shared when there are more than one ISLB in the later.
func (r *Registry) handleNodeAction(action discovery.Action, node discovery.Node) (bool, error) {
	//Add authentication here
	log.Debugf("handleNode: service %v, action %v => id %v, RPC %v", node.Service, action, node.ID(), node.RPC)

	//TODO: Put node info into the redis.
	r.mutex.Lock()
	defer r.mutex.Unlock()

	switch action {
	case discovery.Save:
		fallthrough
	case discovery.Update:
		r.nodes[node.ID()] = node
	case discovery.Delete:
		delete(r.nodes, node.ID())
	}

	return true, nil
}

func (r *Registry) handleGetNodes(service string, params map[string]interface{}) ([]discovery.Node, error) {
	//Add load balancing here.
	log.Infof("Get node by %v, params %v", service, params)

	if service == proto.ServiceSFU {
		nid := "*"
		sid := ""
		if val, ok := params["nid"]; ok {
			nid = val.(string)
		}

		if val, ok := params["sid"]; ok {
			sid = val.(string)
		}

		// find node by nid/sid from reids
		mkey := r.dc + "/" + nid + "/" + sid
		log.Infof("islb.FindNode: mkey => %v", mkey)
		for _, key := range r.redis.Keys(mkey) {
			value := r.redis.Get(key)
			log.Debugf("key: %v, value: %v", key, value)
		}
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	nodesResp := []discovery.Node{}
	for _, item := range r.nodes {
		if item.Service == service || service == "*" {
			nodesResp = append(nodesResp, item)
		}
	}

	return nodesResp, nil
}
