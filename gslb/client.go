package gslb

import (
	"context"
	"errors"
	"time"

	"go.etcd.io/etcd/clientv3"
)

const (
	defaultDialTimeout      = time.Second * 5
	defaultGrantTimeout     = 10
	defaultOperationTimeout = time.Second * 5
)

type WatchCallback func(clientv3.WatchChan)

type Client struct {
	client  *clientv3.Client
	leaseID clientv3.LeaseID
}

func NewClient(endpoints []string, key string, value string) (*Client, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: defaultDialTimeout,
	})

	if err != nil {
		return nil, err
	}
	resp, err := cli.Grant(context.TODO(), defaultGrantTimeout)
	if err != nil {
		return nil, err
	}

	_, err = cli.Put(context.TODO(), key, value, clientv3.WithLease(resp.ID))
	if err != nil {
		return nil, err
	}

	return &Client{
		client:  cli,
		leaseID: resp.ID,
	}, nil
}

func (e *Client) KeepAlive() error {
	_, err := e.client.KeepAlive(context.TODO(), e.leaseID)
	return err
}

func (e *Client) GetAliveID() (int64, error) {
	return int64(e.leaseID), nil
}

func (e *Client) Watch(key string, watchFunc WatchCallback) error {
	if watchFunc == nil {
		return errors.New("watchFunc is nil")
	}
	watchFunc(e.client.Watch(context.Background(), key))

	return nil
}

func (e *Client) Close() error {
	return e.client.Close()
}

func (e *Client) Put(key, value string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	_, err := e.client.Put(ctx, key, value)
	cancel()

	return err
}

func (e *Client) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	resp, err := e.client.Get(ctx, key)
	if err != nil {
		return "", err
	}
	var val string
	for _, ev := range resp.Kvs {
		val = string(ev.Value)
	}
	cancel()

	return val, err
}

func (e *Client) GetByPrefix(key string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultOperationTimeout)
	resp, err := e.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}
	var vals []string
	for _, ev := range resp.Kvs {
		vals = append(vals, string(ev.Value))
	}
	cancel()

	return vals, err
}

func (e *Client) Update(key, value string) error {
	_, err := e.client.Put(context.TODO(), key, value, clientv3.WithLease(e.leaseID))
	return err
}

func (e *Client) PutWithTtl(key, value string, ttl int64) error {
	resp, err := e.client.Grant(context.TODO(), ttl)
	if err != nil {
		return err
	}

	_, err = e.client.Put(context.TODO(), key, value, clientv3.WithLease(resp.ID))
	return err
}
