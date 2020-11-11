package discovery

import (
	"context"
	"encoding/json"
	"path"
	"strings"
	"sync"
	"time"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/util"
	"go.etcd.io/etcd/clientv3"
)

// State define the node state type
type State int32

const (
	// NodeUp node starting up
	NodeUp State = 0
	// NodeDown node shutdown
	NodeDown State = 1

	defautlWatchInterval    = 2 * time.Second
	defaultDialTimeout      = 5 * time.Second
	defaultGrantTimeout     = 5
	defaultOperationTimeout = 5 * time.Second
	defaultWatchRetryDelay  = 2 * time.Second
	defaultKeepRetryDelay   = 2 * time.Second
)

// Node represents a node info
type Node struct {
	DC      string
	Service string
	NID     string
	IP      string
}

// scheme represents the node prefix
func (n *Node) scheme() string {
	return "/" + n.DC + "/node/"
}

// ID return the node id with scheme prefix
func (n *Node) ID() string {
	return "/" + n.DC + "/node/" + n.NID
}

// Service represents a service node
type Service struct {
	node     Node
	nodes    map[string]*Node
	nodeLock *sync.RWMutex
	client   *clientv3.Client
	stop     util.AtomicBool
	down     chan bool
	mutex    sync.Mutex
}

// NewService create a service instance
func NewService(service string, dc string, addrs []string) (*Service, error) {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   addrs,
		DialTimeout: defaultDialTimeout,
	})
	if err != nil {
		return nil, err
	}

	s := Service{
		node: Node{
			DC:      dc,
			Service: service,
			NID:     service + "-" + util.RandomString(12),
			IP:      util.GetInterfaceIP(),
		},
		client: client,
		down:   make(chan bool),
	}

	log.Infof("new node: %v", s.node)

	return &s, nil
}

// Close service
func (s *Service) Close() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.stop.Get() {
		return
	}

	s.stop.Set(true)

	log.Infof("node close: %v", s.node)
	close(s.down)
	s.client.Close()
}

// NID return node id
func (s *Service) NID() string {
	return s.node.NID
}

// KeepAlive service keepalive
func (s *Service) KeepAlive() {
	go s.keepAlive(s.node)
}

// keepAlive service keepalive
func (s *Service) keepAlive(node Node) {
	id := node.ID()

	log.Infof("start keepalive: %s", id)
	defer log.Infof("stop keepalive: %s", id)

	val, err := json.Marshal(node)
	if err != nil {
		log.Errorf("json marshal error: %v, %v", node, err)
		return
	}

	for {
		if s.stop.Get() {
			return
		}

		grant, err := s.client.Grant(context.TODO(), defaultGrantTimeout)
		if err != nil {
			log.Errorf("etcd.Grant error: %s, %v", id, err)
			time.Sleep(defaultKeepRetryDelay)
			continue
		}

		_, err = s.client.Put(context.TODO(), id, string(val), clientv3.WithLease(grant.ID))
		if err != nil {
			log.Errorf("etcd.Put error: %s, %v", id, err)
			time.Sleep(defaultKeepRetryDelay)
			continue
		}
		ch, err := s.client.KeepAlive(context.TODO(), grant.ID)
		if err != nil {
			log.Errorf("etcd.KeepAlive error: %s, %v", id, err)
			time.Sleep(defaultKeepRetryDelay)
			continue
		}

		go func(id string) {
			log.Infof("start receiving keepalive-response: %s", id)
			defer log.Infof("stop receiving keepalive-response: %s", id)

			for {
				if s.stop.Get() {
					return
				}
				// just read, fix etcd-server warning "lease keepalive response queue is full; dropping response send""
				ka, ok := <-ch
				if ok {
					log.Tracef("receive keepalive-response: id=%d, ttl=%d", ka.ID, ka.TTL)
				} else {
					log.Infof("can not receive keepalive-response")
					if !s.stop.Get() {
						s.mutex.Lock()
						log.Infof("node re-registration: %s", id)
						s.down <- true
						s.mutex.Unlock()
					}
					return
				}

				time.Sleep(500 * time.Millisecond)
			}
		}(id)

		log.Infof("node registered successfully: %s", id)

		<-s.down
	}
}

// Watch nodes
func (s *Service) Watch(service string, onStateChange func(state State, node Node)) {
	if s.nodes == nil {
		s.nodes = make(map[string]*Node)
	}
	if s.nodeLock == nil {
		s.nodeLock = new(sync.RWMutex)
	}
	go s.watch(service, onStateChange)
}

// watch nodes
func (s *Service) watch(service string, onStateChange func(state State, node Node)) {
	log.Infof("start watching nodes: %s", service)
	defer log.Infof("stop watching nodes: %s", service)

	key := path.Join(s.node.scheme(), strings.Replace(service, "/", "-", -1))

	for {
		if s.stop.Get() {
			break
		}

		ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
		resp, err := s.client.Get(ctx, key, clientv3.WithPrefix())
		cancel()
		if err != nil {
			log.Errorf("etcd.Get error: %v", err)
			time.Sleep(defaultWatchRetryDelay)
			continue
		}

		for _, kv := range resp.Kvs {
			var node Node
			if err := json.Unmarshal(kv.Value, &node); err != nil {
				continue
			}
			//id := string(n.Key)
			id := node.ID()
			if s.getNode(id) != nil {
				continue
			}
			log.Infof("node up: %s", id)
			s.addNode(id, node)
			onStateChange(NodeUp, node)
			s.handleRespone(id, onStateChange)
		}

		time.Sleep(defautlWatchInterval)
	}

	// self node stop
	id := s.node.ID()
	if node := s.getNode(id); node != nil {
		s.delNode(id)
		onStateChange(NodeDown, *node)
	}
}

// handleRespone handle watch-respone
func (s *Service) handleRespone(id string, onStateChange func(state State, nodes Node)) {
	go func(id string) {
		log.Infof("start handling watch-response: %s", id)
		defer log.Infof("stop handling watch-response: %s", id)

		ch := s.client.Watch(context.Background(), id, clientv3.WithPrefix())
		for {
			resp, ok := <-ch
			if ok {
				log.Tracef("receive watch-respone: %v", resp.Events)
			} else {
				log.Infof("can not receive watch-response")
				return
			}
			if resp.Canceled {
				log.Infof("etcd.Watch canceled: %s, %s", id, resp.Err())
				return
			}
			for _, e := range resp.Events {
				log.Infof("watching event: %s, %q, %q", e.Type, e.Kv.Key, e.Kv.Value)
				if id != string(e.Kv.Key) {
					log.Errorf("node id mismatching: %s, %s", id, e.Kv.Key)
					continue
				}
				if e.Type == clientv3.EventTypeDelete {
					log.Infof("node dwon: %s", id)
					if node := s.getNode(id); node != nil {
						s.delNode(id)
						onStateChange(NodeDown, *node)
					}
					return
				}
			}
		}
	}(id)
}

// addNode add a node
func (s *Service) addNode(id string, node Node) {
	s.nodeLock.Lock()
	defer s.nodeLock.Unlock()

	s.nodes[id] = &node
}

// getNode get node by id
func (s *Service) getNode(id string) *Node {
	s.nodeLock.RLock()
	defer s.nodeLock.RUnlock()

	return s.nodes[id]
}

// delNode delete the node
func (s *Service) delNode(id string) {
	s.nodeLock.Lock()
	defer s.nodeLock.Unlock()

	delete(s.nodes, id)
}
