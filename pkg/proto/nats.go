package proto

import (
	"bytes"
	"encoding/gob"
	"errors"
	"time"

	log "github.com/pion/ion-log"

	"github.com/nats-io/nats.go"
)

// rpcmsg is a structure used by Subscribe and Publish.
type rpcmsg struct {
	Data interface{}
}

// RPCError represents a error string for rpc
type RPCError struct {
	Err string
}

// newError create a RPCError instanse
func newError(err string) *RPCError {
	return &RPCError{err}
}

// MsgHandler is a callback function that processes messages delivered to
// asynchronous subscribers.
type MsgHandler func(msg interface{}) (interface{}, error)

// NatsRPC represents a rpc base nats
type NatsRPC struct {
	nc *nats.Conn
}

// NewNatsRPC create a instanse and connect to nats server.
func NewNatsRPC(urls string) (*NatsRPC, error) {
	r := &NatsRPC{}
	err := r.Connect(urls)
	return r, err
}

// Connect to nats server.
func (r *NatsRPC) Connect(url string) error {
	// connect options
	opts := []nats.Option{nats.Name("nats ion service")}
	opts = r.setupConnOptions(opts)

	// connect to nats server
	var err error
	if r.nc, err = nats.Connect(url, opts...); err != nil {
		return err
	}

	return nil
}

// Close the connection to the server.
func (r *NatsRPC) Close() {
	r.nc.Close()
}

func (r *NatsRPC) setupConnOptions(opts []nats.Option) []nats.Option {
	totalWait := 10 * time.Minute
	reconnectDelay := time.Second

	opts = append(opts, nats.ReconnectWait(reconnectDelay))
	opts = append(opts, nats.MaxReconnects(int(totalWait/reconnectDelay)))
	opts = append(opts, nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
		if !nc.IsClosed() {
			log.Infof("Disconnected due to: %s, will attempt reconnects for %.0fm", err, totalWait.Minutes())
		}
	}))
	opts = append(opts, nats.ReconnectHandler(func(nc *nats.Conn) {
		log.Infof("Reconnected [%s]", nc.ConnectedUrl())
	}))
	opts = append(opts, nats.ClosedHandler(func(nc *nats.Conn) {
		if !nc.IsClosed() {
			log.Errorf("Exiting: no servers available")
		} else {
			log.Errorf("Exiting")
		}
	}))
	return opts
}

// Subscribe will express interest in the given subject.
// Messages will be delivered to the associated MsgHandler.
func (r *NatsRPC) Subscribe(subj string, handle MsgHandler) (*nats.Subscription, error) {
	return r.QueueSubscribe(subj, "", handle)
}

// QueueSubscribe creates an asynchronous queue subscriber on the given subject.
// All subscribers with the same queue name will form the queue group and
// only one member of the group will be selected to receive any given
// message asynchronously.
func (r *NatsRPC) QueueSubscribe(subj, queue string, handle MsgHandler) (*nats.Subscription, error) {
	return r.nc.QueueSubscribe(subj, queue, func(msg *nats.Msg) {
		var got rpcmsg
		if err := Unmarshal(msg.Data, &got); err != nil {
			log.Errorf("decode msg error: %v", err)
		}

		result, err := handle(got.Data)
		if err != nil {
			result = newError(err.Error())
		}

		resp := &rpcmsg{Data: result}

		if msg.Reply != "" {
			data, err := Marshal(resp)
			if err != nil {
				log.Errorf("marshal error: %v", err)
			}
			if err := msg.Respond(data); err != nil {
				log.Errorf("respond error: %v", err)
			}
		}
	})
}

// Request will send a request payload and deliver the response message,
// or an error, including a timeout if no message was received properly.
func (r *NatsRPC) Request(subj string, data interface{}) (interface{}, error) {
	d, err := Marshal(&rpcmsg{Data: data})
	if err != nil {
		return nil, err
	}

	resp, err := r.nc.Request(subj, d, nats.DefaultTimeout)
	if err != nil {
		return nil, err
	}

	var result rpcmsg
	err = Unmarshal(resp.Data, &result)
	if err != nil {
		return nil, err
	}

	if v, ok := result.Data.(*RPCError); ok {
		err = errors.New(v.Err)
		result.Data = nil
	}

	return result.Data, err
}

// Publish publishes the data argument to the given subject. The data
// argument is left untouched and needs to be correctly interpreted on
// the receiver.
func (r *NatsRPC) Publish(subj string, data interface{}) error {
	d, err := Marshal(&rpcmsg{
		Data: data,
	})
	if err != nil {
		return err
	}
	return r.nc.Publish(subj, d)
}

func init() {
	gob.Register(&RPCError{})
}

// Unmarshal parses the encoded data and stores the result
// in the value pointed to by v
func Unmarshal(data []byte, v interface{}) error {
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	return dec.Decode(v)
}

// Marshal encodes v and returns encoded data
func Marshal(v interface{}) ([]byte, error) {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(v)
	if err != nil {
		return []byte{}, err
	}
	return buf.Bytes(), nil
}
