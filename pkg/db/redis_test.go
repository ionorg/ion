package db

import (
	"testing"
	"time"
)

func NewRedisSingle() *Redis {
	r := NewRedis(Config{
		Addrs: []string{
			":6379",
		},
		DB:  0,
		Pwd: "",
	})
	return r
}

func TestNew_Close(t *testing.T) {
	r := NewRedisSingle()
	defer r.Close()
}

func TestAcquire_Release(t *testing.T) {
	r := NewRedisSingle()
	defer r.Close()

	lockKey := "123"
	r.Acquire(lockKey)
	res := r.Get("lock-" + lockKey)
	if res != "1" {
		t.Errorf("acquire error")
	}
	t.Log("acquire ok")

	r.Release(lockKey)
	res = r.Get("lock-" + lockKey)
	if res != "" {
		t.Errorf("release error")
	}
	t.Log("release ok")
}

func TestConcurrencyAcquire(t *testing.T) {
	r := NewRedisSingle()
	defer r.Close()
	lockKey := "123"
	t.Log("A acquire")
	r.Acquire(lockKey)

	go func() {
		t.Log("B try acquire")
		r.Acquire(lockKey)
		t.Log("B acquire ok")
		r.Release(lockKey)
		t.Log("B release")
	}()

	time.Sleep(time.Second * 2)
	t.Log("A wait 2 second")
	r.Release(lockKey)
	t.Log("A release ok")
	time.Sleep(time.Second * 2)
}

func TestSet_Get_Del(t *testing.T) {
	r := NewRedisSingle()
	defer r.Close()
	key, value := "key", "value"
	err := r.Set(key, value, time.Second)
	if err != nil {
		t.Error("Set error")
	}
	res := r.Get(key)
	if res != value {
		t.Error("Get error")
	}
	err = r.Del(key)
	if err != nil {
		t.Error("Del error")
	}
	res = r.Get(key)
	if res != "" {
		t.Error("Get error")
	}
}

func TestHSet_HGet_HDel(t *testing.T) {
	r := NewRedisSingle()
	defer r.Close()
	key, field, value := "key", "field", "value"
	err := r.HSet(key, field, value)
	if err != nil {
		t.Error("HSet error:", err)
	}

	res := r.HGet(key, field)
	if res != value {
		t.Error("HGet error:", err)
	}

	err = r.HDel(key, field)
	if err != nil {
		t.Error("HDel error:", err)
	}
}

func TestHMSet_HMGet_HGetAll_HDel(t *testing.T) {
	r := NewRedisSingle()
	defer r.Close()
	key, field1, value1, field2, value2 := "key", "field1", "value1", "field2", "value2"
	err := r.HMSet(key, field1, value1, field2, value2)
	if err != nil {
		t.Error("HMSet error:", err)
	}

	res := r.HMGet(key, field1, field2)
	if res[0] != value1 && res[1] != value2 {
		t.Error("HMGet error:", err)
	}

	resMap := r.HGetAll(key)
	if resMap[field1] != value1 && resMap[field2] != value2 {
		t.Error("HGetAll error:", err)
	}

	err = r.Del(key)
	if err != nil {
		t.Error("Del error:", err)
	}
}

func TestHSetTTL_HMSetTTL(t *testing.T) {
	r := NewRedisSingle()
	defer r.Close()
	key, field, value := "key", "field", "value"
	err := r.HSetTTL(time.Second, key, field, value)
	if err != nil {
		t.Error("HSetTTL error:", err)
	}

	res := r.HGet(key, field)
	if res != value {
		t.Error("HGet error:", err)
	}
	time.Sleep(time.Second * 2)

	res = r.HGet(key, field)
	if res == value {
		t.Error("HGet error:", err)
	}

	err = r.HDel(key, field)
	if err != nil {
		t.Error("HDel error:", err)
	}
}
