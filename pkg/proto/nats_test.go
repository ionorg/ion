package proto

import (
	"encoding/gob"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
)

func TestRequestString(t *testing.T) {
	n, err := NewNatsRPC(nats.DefaultURL)
	assert.NoError(t, err)
	defer n.Close()

	sub, err := n.Subscribe("rpc-id", func(msg interface{}) (interface{}, error) {
		fmt.Printf("request: %+v\n", msg)
		assert.Equal(t, msg, "tommy")
		return "joined", nil
	})
	assert.NoError(t, err)

	resp, err := n.Request("rpc-id", "tommy")
	fmt.Printf("request: resp=%+v, err=%+v\n", resp, err)
	assert.NoError(t, err)
	assert.Equal(t, resp, "joined")

	err = sub.Unsubscribe()
	assert.NoError(t, err)
}

func TestRequestStruct(t *testing.T) {
	n, err := NewNatsRPC(nats.DefaultURL)
	assert.NoError(t, err)
	defer n.Close()

	type JoinReq struct {
		ID   int
		Name string
	}

	type JoinResp struct {
		MID string
	}

	gob.Register(&JoinReq{})
	gob.Register(&JoinResp{})

	reqMsg := &JoinReq{ID: 1234, Name: "tommy"}
	respMsg := &JoinResp{MID: "mid-1234"}

	sub, err := n.Subscribe("rpc-id", func(msg interface{}) (interface{}, error) {
		fmt.Printf("request: %+v\n", msg)
		assert.Equal(t, msg, reqMsg)
		return respMsg, nil
	})
	assert.NoError(t, err)

	resp, err := n.Request("rpc-id", reqMsg)
	fmt.Printf("respone: resp=%+v, err=%+v\n", resp, err)
	assert.NoError(t, err)
	assert.Equal(t, resp, respMsg)

	err = sub.Unsubscribe()
	assert.NoError(t, err)
}

func TestRequestNil(t *testing.T) {
	n, err := NewNatsRPC(nats.DefaultURL)
	assert.NoError(t, err)
	defer n.Close()

	sub, err := n.Subscribe("rpc-id", func(msg interface{}) (interface{}, error) {
		fmt.Printf("request: %+v\n", msg)
		assert.Equal(t, msg, nil)
		return nil, nil
	})
	assert.NoError(t, err)

	resp, err := n.Request("rpc-id", nil)
	fmt.Printf("request: resp=%+v, err=%+v\n", resp, err)
	assert.NoError(t, err)
	assert.Equal(t, resp, nil)

	err = sub.Unsubscribe()
	assert.NoError(t, err)
}

func TestPublish(t *testing.T) {
	n, err := NewNatsRPC(nats.DefaultURL)
	assert.NoError(t, err)
	defer n.Close()

	done := make(chan bool)

	sub, err := n.Subscribe("rpc-id", func(msg interface{}) (interface{}, error) {
		fmt.Printf("request: %+v\n", msg)
		assert.Equal(t, msg, "tommy")
		done <- true
		return nil, nil
	})
	assert.NoError(t, err)

	err = sub.AutoUnsubscribe(1)
	assert.NoError(t, err)

	err = n.Publish("rpc-id", "tommy")
	assert.NoError(t, err)

	select {
	case <-done:
		return
	case <-time.After(time.Second):
		t.Error("request timeout")
		return
	}
}

func TestRequestError(t *testing.T) {
	n, err := NewNatsRPC(nats.DefaultURL)
	assert.NoError(t, err)
	defer n.Close()

	sub, err := n.Subscribe("rpc-id", func(msg interface{}) (interface{}, error) {
		fmt.Printf("request: %+v\n", msg)
		assert.Equal(t, msg, "tommy")
		return nil, errors.New("join faild")
	})
	assert.NoError(t, err)

	resp, err := n.Request("rpc-id", "tommy")
	fmt.Printf("request: resp=%+v, err=%+v\n", resp, err)
	assert.Equal(t, resp, nil)
	assert.Equal(t, err.Error(), "join faild")

	err = sub.Unsubscribe()
	assert.NoError(t, err)
}
