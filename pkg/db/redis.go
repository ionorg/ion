package db

import (
	"context"
	"fmt"
	"time"

	log "github.com/pion/ion-log"

	"github.com/go-redis/redis/v7"
)

var (
	lockExpire = 3 * time.Second
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
		return r
	}

	r.cluster = redis.NewClusterClient(
		&redis.ClusterOptions{
			Addrs:        c.Addrs,
			Password:     c.Pwd,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
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
	r.Acquire(k)
	defer r.Release(k)
	if r.clusterMode {
		return r.cluster.Set(k, v, t).Err()
	}
	return r.single.Set(k, v, t).Err()
}

func (r *Redis) Get(k string) interface{} {
	r.Acquire(k)
	defer r.Release(k)
	if r.clusterMode {
		return r.cluster.Get(k).Val()
	}
	return r.single.Get(k).Val()
}

func (r *Redis) HSet(key, field string, value interface{}) error {
	r.Acquire(key)
	defer r.Release(key)
	if r.clusterMode {
		return r.cluster.HSet(key, field, value).Err()
	}
	return r.single.HSet(key, field, value).Err()
}

func (r *Redis) HMSet(key, field string, values ...interface{}) error {
	r.Acquire(key)
	defer r.Release(key)
	if r.clusterMode {
		return r.cluster.HMSet(key, field, values).Err()
	}
	return r.single.HSet(key, field, values).Err()
}

func (r *Redis) HGet(key, field string) string {
	r.Acquire(key)
	defer r.Release(key)
	if r.clusterMode {
		return r.cluster.HGet(key, field).Val()
	}
	return r.single.HGet(key, field).Val()
}

func (r *Redis) HGetAll(key string) map[string]string {
	r.Acquire(key)
	defer r.Release(key)
	if r.clusterMode {
		return r.cluster.HGetAll(key).Val()
	}
	return r.single.HGetAll(key).Val()
}

func (r *Redis) HDel(key, field string) error {
	r.Acquire(key)
	defer r.Release(key)
	if r.clusterMode {
		return r.cluster.HDel(key, field).Err()
	}
	return r.single.HDel(key, field).Err()
}

func (r *Redis) Expire(key string, t time.Duration) error {
	r.Acquire(key)
	defer r.Release(key)
	if r.clusterMode {
		return r.cluster.Expire(key, t).Err()
	}

	return r.single.Expire(key, t).Err()
}

func (r *Redis) HSetTTL(t time.Duration, key, field string, value interface{}) error {
	r.Acquire(key)
	defer r.Release(key)
	if r.clusterMode {
		if err := r.cluster.HSet(key, field, value).Err(); err != nil {
			return err
		}
		return r.cluster.Expire(key, t).Err()
	}
	if err := r.single.HSet(key, field, value).Err(); err != nil {
		return err
	}
	return r.single.Expire(key, t).Err()
}

func (r *Redis) HMSetTTL(t time.Duration, k string, values ...interface{}) error {
	r.Acquire(k)
	defer r.Release(k)
	if r.clusterMode {
		if err := r.cluster.HMSet(k, values...).Err(); err != nil {
			return err
		}
		return r.cluster.Expire(k, t).Err()
	}

	if err := r.single.HMSet(k, values...).Err(); err != nil {
		return err
	}
	return r.single.Expire(k, t).Err()
}

func (r *Redis) Keys(key string) []string {
	r.Acquire(key)
	defer r.Release(key)
	if r.clusterMode {
		return r.cluster.Keys(key).Val()
	}
	return r.single.Keys(key).Val()
}

func (r *Redis) Del(k string) error {
	r.Acquire(k)
	defer r.Release(k)
	if r.clusterMode {
		return r.cluster.Del(k).Err()
	}
	return r.single.Del(k).Err()
}

// Watch http://redisdoc.com/topic/notification.html
func (r *Redis) Watch(ctx context.Context, key string) <-chan interface{} {
	r.Acquire(key)
	defer r.Release(key)
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

func (r *Redis) lock(key string) bool {
	// Tips: use ("lock-"+key) as lock key is better than (key+"-lock")
	// this avoid "keys /xxxxx/*" to get this lock
	if r.clusterMode {
		ok, _ := r.cluster.SetNX("lock-"+key, 1, 3*time.Second).Result()
		return ok
	}
	ok, _ := r.single.SetNX("lock-"+key, 1, 3*time.Second).Result()
	return ok
}

func (r *Redis) unlock(key string) {
	if r.clusterMode {
		r.cluster.Del("lock-" + key).Result()
	}
	r.single.Del("lock-" + key).Result()
}

// Acquire a destributed lock
func (r *Redis) Acquire(key string) {
	// retry if lock failed
	for !r.lock(key) {
		time.Sleep(time.Millisecond * 100)
	}
}

// Release a destributed lock
func (r *Redis) Release(key string) {
	r.unlock(key)
}
