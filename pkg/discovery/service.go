package discovery

import (
	"context"
	"encoding/json"
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
	node   Node
	client *clientv3.Client
	stop   util.AtomicBool
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
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
	}

	s.ctx, s.cancel = context.WithCancel(context.Background())

	log.Infof("new node: %v", s.node)

	return &s, nil
}

// Close service
func (s *Service) Close() {
	if s.stop.Get() {
		return
	}
	s.stop.Set(true)

	log.Infof("node close: %v", s.node)
	s.cancel()
	s.wg.Wait()
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
	s.wg.Add(1)
	defer s.wg.Done()

	id := node.ID()
	log.Infof("start keepalive: %s", id)
	defer log.Infof("stop keepalive: %s", id)

	val, err := json.Marshal(node)
	if err != nil {
		log.Errorf("json marshal error: %v, %v", node, err)
		return
	}

	for {
		resp, err := s.client.Grant(s.ctx, defaultGrantTimeout)
		if err != nil {
			log.Errorf("etcd.Grant error: %s, %v", id, err)
			time.Sleep(defaultKeepRetryDelay)
			continue
		}
		leaseID := resp.ID

		_, err = s.client.Put(s.ctx, id, string(val), clientv3.WithLease(leaseID))
		if err != nil {
			log.Errorf("etcd.Put error: %s, %v", id, err)
			time.Sleep(defaultKeepRetryDelay)
			continue
		}
		ch, err := s.client.KeepAlive(s.ctx, leaseID)
		if err != nil {
			log.Errorf("etcd.KeepAlive error: %s, %v", id, err)
			time.Sleep(defaultKeepRetryDelay)
			continue
		}

		log.Infof("node registered successfully: %s", id)

		done := make(chan bool)
		go func(id string) {
			log.Infof("start receiving keepalive-response: %s, %d", id, leaseID)
			defer log.Infof("stop receiving keepalive-response: %s, %d", id, leaseID)

			for {
				// just read, fix etcd-server warning "lease keepalive response queue is full; dropping response send""
				ka, ok := <-ch
				if ok {
					log.Tracef("receive keepalive-response: id=%d, ttl=%d", ka.ID, ka.TTL)
				} else {
					log.Infof("can not receive keepalive-response")
					break
				}
				if s.stop.Get() {
					break
				}
				time.Sleep(500 * time.Millisecond)
			}

			done <- true
		}(id)
		<-done

		_, err = s.client.Revoke(context.TODO(), leaseID)
		if err != nil {
			log.Errorf("etcd.Revoke error: %s, %d, %v", id, leaseID, err)
		}

		if s.stop.Get() {
			break
		}
	}
}

// Watch nodes
func (s *Service) Watch(service string, onStateChange func(state State, id string, node *Node)) {
	go s.watch(service, onStateChange)
}

// watch the service nodes
func (s *Service) watch(service string, onStateChange func(state State, id string, node *Node)) {
	go func(key string) {
		s.wg.Add(1)
		defer func() {
			id := s.node.ID()
			log.Infof("node down: %s", id)
			onStateChange(NodeDown, id, nil)
			s.wg.Done()
		}()

		log.Infof("start watching: %s", key)
		defer log.Infof("stop watching: %s", key)

		ch := s.client.Watch(s.ctx, key, clientv3.WithPrefix())
		for {
			resp, ok := <-ch
			if ok {
				log.Tracef("receive watch-respone: %v", resp.Events)
			} else {
				log.Infof("can not receive watch-response")
				return
			}
			if resp.Canceled {
				log.Infof("etcd.Watch canceled: %s, %s", key, resp.Err())
				return
			}
			for _, e := range resp.Events {
				log.Infof("watching event: %s, %s, %s", e.Type, e.Kv.Key, e.Kv.Value)
				id := string(e.Kv.Key)
				switch e.Type {
				case clientv3.EventTypePut:
					var node Node
					if err := json.Unmarshal(e.Kv.Value, &node); err != nil {
						log.Warnf("json.Unmarshal error: %v", e.Kv.Value)
						continue
					}
					log.Infof("node up: %s, %+v", id, node)
					onStateChange(NodeUp, id, &node)
				case clientv3.EventTypeDelete:
					log.Infof("node down: %s", id)
					onStateChange(NodeDown, id, nil)
				}
			}
		}
	}(s.node.scheme() + service)
}

// GetNodes get the service nodes
func (s *Service) GetNodes(service string, nodes map[string]Node) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	resp, err := s.client.Get(ctx, s.node.scheme()+service, clientv3.WithPrefix())
	cancel()
	if err != nil {
		return err
	}

	for _, kv := range resp.Kvs {
		var node Node
		if err := json.Unmarshal(kv.Value, &node); err != nil {
			continue
		}
		id := string(kv.Key)
		nodes[id] = node
	}

	return nil
}
