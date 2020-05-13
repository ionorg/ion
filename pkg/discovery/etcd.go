package discovery

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/pion/ion/pkg/log"
	"go.etcd.io/etcd/clientv3"
)

const (
	defaultDialTimeout      = time.Second * 5
	defaultGrantTimeout     = 5
	defaultOperationTimeout = time.Second * 5
)

type WatchCallback func(clientv3.WatchChan)

type Etcd struct {
	client        *clientv3.Client
	liveKeyID     map[string]clientv3.LeaseID
	liveKeyIDLock sync.RWMutex
	stop          bool
}

func newEtcd(endpoints []string) (*Etcd, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: defaultDialTimeout,
	})

	if err != nil {
		log.Errorf("newEtcd err=%v", err)
		return nil, err
	}

	return &Etcd{
		client:    cli,
		liveKeyID: make(map[string]clientv3.LeaseID),
	}, nil
}

func (e *Etcd) keep(key, value string) error {
	resp, err := e.client.Grant(context.TODO(), defaultGrantTimeout)
	if err != nil {
		log.Errorf("Etcd.keep Grant %s %v", key, err)
		return err
	}
	_, err = e.client.Put(context.TODO(), key, value, clientv3.WithLease(resp.ID))
	if err != nil {
		log.Errorf("Etcd.keep Put %s %v", key, err)
		return err
	}

	ch, err := e.client.KeepAlive(context.TODO(), resp.ID)
	if err != nil {
		log.Errorf("Etcd.keep %s %v", key, err)
		return err
	}
	go func() {
		for {
			if e.stop {
				return
			}

			//just read, fix etcd-server warning "lease keepalive response queue is full; dropping response send""
			<-ch
			time.Sleep(time.Millisecond * 100)
		}
	}()
	e.liveKeyIDLock.Lock()
	e.liveKeyID[key] = resp.ID
	e.liveKeyIDLock.Unlock()
	log.Infof("Etcd.keep %s %v %v", key, value, err)
	return nil
}

func (e *Etcd) del(key string, prefix bool) error {
	e.liveKeyIDLock.Lock()
	delete(e.liveKeyID, key)
	e.liveKeyIDLock.Unlock()
	var err error
	if prefix {
		_, err = e.client.Delete(context.TODO(), key, clientv3.WithPrefix())
	} else {
		_, err = e.client.Delete(context.TODO(), key)
	}
	return err
}

func (e *Etcd) watch(key string, watchFunc WatchCallback, prefix bool) error {
	if watchFunc == nil {
		return errors.New("watchFunc is nil")
	}
	if prefix {
		watchFunc(e.client.Watch(context.Background(), key, clientv3.WithPrefix()))
	} else {
		watchFunc(e.client.Watch(context.Background(), key))
	}

	return nil
}

// func (e *Etcd) close() error {
// 	if e.stop {
// 		return errors.New("Etcd already close")
// 	}
// 	e.stop = true
// 	e.liveKeyIDLock.Lock()
// 	for k := range e.liveKeyID {
// 		_, err := e.client.Delete(context.TODO(), k)
// 		if err != nil {
// 			log.Errorf("e.client.Delete %v", err)
// 		}
// 	}
// 	e.liveKeyIDLock.Unlock()
// 	return e.client.Close()
// }

// func (e *Etcd) put(key, value string) error {
// ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
// _, err := e.client.Put(ctx, key, value)
// cancel()

// return err
// }

func (e *Etcd) get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	resp, err := e.client.Get(ctx, key)
	if err != nil {
		cancel()
		return "", err
	}
	var val string
	for _, ev := range resp.Kvs {
		val = string(ev.Value)
	}
	cancel()

	return val, err
}

func (e *Etcd) getByPrefix(key string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	resp, err := e.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		cancel()
		return nil, err
	}
	m := make(map[string]string)
	for _, kv := range resp.Kvs {
		m[string(kv.Key)] = string(kv.Value)
	}
	cancel()

	return m, err
}

// GetResponseByPrefix .
func (e *Etcd) GetResponseByPrefix(key string) (*clientv3.GetResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	defer cancel()
	resp, err := e.client.Get(ctx, key, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
	// try again if time out
	if err == context.DeadlineExceeded {
		ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout*2)
		defer cancel()
		resp, err := e.client.Get(ctx, key, clientv3.WithPrefix(), clientv3.WithSort(clientv3.SortByKey, clientv3.SortDescend))
		return resp, err
	}
	return resp, err
}

func (e *Etcd) update(key, value string) error {
	e.liveKeyIDLock.Lock()
	id := e.liveKeyID[key]
	e.liveKeyIDLock.Unlock()
	_, err := e.client.Put(context.TODO(), key, value, clientv3.WithLease(id))
	if err != nil {
		err = e.keep(key, value)
		if err != nil {
			log.Errorf("Etcd.Keep %s %s %v", key, value, err)
		}
	}
	// log.Infof("Etcd.Update %s %s %v", key, value, err)
	return err
}
