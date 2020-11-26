package db

import (
	"context"
	"fmt"
	"sync"
	"time"

	log "github.com/pion/ion-log"

	"github.com/go-redis/redis/v7"
)

type Config struct {
	Addrs []string `mapstructure:"addrs"`
	Pwd   string   `mapstructure:"password"`
	DB    int      `mapstructure:"db"`
}

type Redis struct {
	cluster     *redis.ClusterClient
	single      *redis.Client
	clusterMode bool
	mutex       *sync.Mutex
}

func NewRedis(c Config) *Redis {
	if len(c.Addrs) == 0 {
		return nil
	}

	r := &Redis{}
	if len(c.Addrs) == 1 {
		r.single = redis.NewClient(
			&redis.Options{
				Addr:         c.Addrs[0], // use default Addr
				Password:     c.Pwd,      // no password set
				DB:           c.DB,       // use default DB
				DialTimeout:  3 * time.Second,
				ReadTimeout:  5 * time.Second,
				WriteTimeout: 5 * time.Second,
			})
		if err := r.single.Ping().Err(); err != nil {
			log.Errorf(err.Error())
			return nil
		}
		r.single.Do("CONFIG", "SET", "notify-keyspace-events", "AKE")
		r.clusterMode = false
		r.mutex = new(sync.Mutex)
		return r
	}

	r.cluster = redis.NewClusterClient(
		&redis.ClusterOptions{
			Addrs:        c.Addrs,
			Password:     c.Pwd,
			DialTimeout:  3 * time.Second,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 5 * time.Second,
		})
	if err := r.cluster.Ping().Err(); err != nil {
		log.Errorf(err.Error())
	}
	r.cluster.Do("CONFIG", "SET", "notify-keyspace-events", "AKE")
	r.clusterMode = true
	return r
}

func (r *Redis) Close() {
	if r.single != nil {
		r.single.Close()
	}
	if r.cluster != nil {
		r.cluster.Close()
	}
}

func (r *Redis) Set(k, v string, t time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.clusterMode {
		return r.cluster.Set(k, v, t).Err()
	}
	return r.single.Set(k, v, t).Err()
}

func (r *Redis) Get(k string) interface{} {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.clusterMode {
		return r.cluster.Get(k).Val()
	}
	return r.single.Get(k).Val()
}

func (r *Redis) HSet(k, field string, value interface{}) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.clusterMode {
		return r.cluster.HSet(k, field, value).Err()
	}
	return r.single.HSet(k, field, value).Err()
}

func (r *Redis) HGet(k, field string) string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.clusterMode {
		return r.cluster.HGet(k, field).Val()
	}
	return r.single.HGet(k, field).Val()
}

func (r *Redis) HGetAll(k string) map[string]string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.clusterMode {
		return r.cluster.HGetAll(k).Val()
	}
	return r.single.HGetAll(k).Val()
}

func (r *Redis) HDel(k, field string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.clusterMode {
		return r.cluster.HDel(k, field).Err()
	}
	return r.single.HDel(k, field).Err()
}

func (r *Redis) Expire(k string, t time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.clusterMode {
		return r.cluster.Expire(k, t).Err()
	}

	return r.single.Expire(k, t).Err()
}

func (r *Redis) HSetTTL(k, field string, value interface{}, t time.Duration) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.clusterMode {
		if err := r.cluster.HSet(k, field, value).Err(); err != nil {
			return err
		}
		return r.cluster.Expire(k, t).Err()
	}
	if err := r.single.HSet(k, field, value).Err(); err != nil {
		return err
	}
	return r.single.Expire(k, t).Err()
}

func (r *Redis) Keys(k string) []string {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.clusterMode {
		return r.cluster.Keys(k).Val()
	}
	return r.single.Keys(k).Val()
}

func (r *Redis) Del(k string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	if r.clusterMode {
		return r.cluster.Del(k).Err()
	}
	return r.single.Del(k).Err()
}

// Watch http://redisdoc.com/topic/notification.html
func (r *Redis) Watch(ctx context.Context, key string) <-chan interface{} {
	var pubsub *redis.PubSub
	if r.clusterMode {
		pubsub = r.cluster.PSubscribe(fmt.Sprintf("__key*__:%s", key))
	} else {
		pubsub = r.single.PSubscribe(fmt.Sprintf("__key*__:%s", key))
	}

	res := make(chan interface{})
	go func() {
		for {
			select {
			case msg := <-pubsub.Channel():
				op := msg.Payload
				log.Infof("key => %s, op => %s", key, op)
				res <- op
			case <-ctx.Done():
				pubsub.Close()
				close(res)
				return
			}
		}
	}()

	return res
}
