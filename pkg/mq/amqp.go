package mq

import (
	"encoding/json"
	"net"
	"time"

	"github.com/chuckpreslar/emission"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/util"
	"github.com/streadway/amqp"
)

const (
	connTimeout         = 3 * time.Second
	broadCastRoutingKey = "broadcast"
	rpcExchange         = "rpcExchange"
	broadCastExchange   = "broadCastExchange"
)

type Amqp struct {
	emission.Emitter
	conn             *amqp.Connection
	rpcChannel       *amqp.Channel
	broadCastChannel *amqp.Channel
	rpcQueue         amqp.Queue
	broadCastQueue   amqp.Queue
}

func New(id, url string) *Amqp {
	a := &Amqp{
		Emitter: *emission.NewEmitter(),
	}
	var err error
	a.conn, err = amqp.DialConfig(url, amqp.Config{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, connTimeout)
		},
	})
	if err != nil {
		log.Panicf(err.Error())
		return nil
	}

	err = a.initRPC(id)
	if err != nil {
		log.Panicf(err.Error())
		return nil
	}

	err = a.initBroadCast()
	if err != nil {
		log.Panicf(err.Error())
		return nil
	}
	return a
}

func (a *Amqp) Close() {
	if a.conn != nil {
		a.conn.Close()
	}
}

func (a *Amqp) initRPC(id string) error {
	var err error
	a.rpcChannel, err = a.conn.Channel()
	if err != nil {
		return err
	}

	// a direct router
	err = a.rpcChannel.ExchangeDeclare(rpcExchange, "direct", true, false, false, false, nil)

	// a receive queue
	a.rpcQueue, err = a.rpcChannel.QueueDeclare(id, false, false, true, false, nil)

	if err != nil {
		return err
	}

	// bind queue to it's name
	err = a.rpcChannel.QueueBind(a.rpcQueue.Name, a.rpcQueue.Name, rpcExchange, false, nil)
	return err
}

func (a *Amqp) initBroadCast() error {
	var err error
	a.broadCastChannel, err = a.conn.Channel()
	if err != nil {
		return err
	}

	// a receiving broadcast msg queue
	err = a.broadCastChannel.ExchangeDeclare("broadCastExchange", "topic", true, false, false, false, nil)

	a.broadCastQueue, err = a.broadCastChannel.QueueDeclare("", false, false, true, false, nil)

	if err != nil {
		return err
	}

	// bind to broadCastRoutingKey
	err = a.broadCastChannel.QueueBind(a.broadCastQueue.Name, broadCastRoutingKey, broadCastExchange, false, nil)
	return err
}

func (a *Amqp) ConsumeRPC() (<-chan amqp.Delivery, error) {
	return a.rpcChannel.Consume(
		a.rpcQueue.Name, // queue
		"",              // consumer
		true,            // auto ack
		false,           // exclusive
		false,           // no local
		false,           // no wait
		nil,             // args
	)
}

func (a *Amqp) ConsumeBroadcast() (<-chan amqp.Delivery, error) {
	return a.broadCastChannel.Consume(
		a.broadCastQueue.Name, // queue
		"",                    // consumer
		true,                  // auto ack
		false,                 // exclusive
		false,                 // no local
		false,                 // no wait
		nil,                   // args
	)
}

func (a *Amqp) RpcCall(id string, msg map[string]interface{}, corrID string) (string, error) {
	str := util.Marshal(msg)
	correlatioID := ""
	if corrID == "" {
		correlatioID = util.RandStr(8)
	} else {
		correlatioID = corrID
	}
	err := a.rpcChannel.Publish(
		rpcExchange, // exchange
		id,          // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: correlatioID,
			ReplyTo:       a.rpcQueue.Name,
			Body:          []byte(str),
		})
	if err != nil {
		return "", err
	}
	return correlatioID, nil
}

func (a *Amqp) RpcCallWithResp(id string, msg map[string]interface{}, callback interface{}) error {
	str := util.Marshal(msg)
	corrID := util.RandStr(8)
	a.Emitter.On(corrID, callback)
	log.Debugf("Amqp.RpcCallWithResp id=%s msg=%v corrID=%s callback=%v", id, msg, corrID, callback)
	err := a.rpcChannel.Publish(
		rpcExchange, // exchange
		id,          // routing key
		false,       // mandatory
		false,       // immediate
		amqp.Publishing{
			ContentType:   "text/plain",
			CorrelationId: corrID,
			ReplyTo:       a.rpcQueue.Name,
			Body:          []byte(str),
		})
	if err != nil {
		return err
	}
	return nil
}

func (a *Amqp) BroadCast(msg map[string]interface{}) error {
	str, err := json.Marshal(msg)
	if err != nil {
		log.Errorf("Marshal %v", err)
		return err
	}

	return a.broadCastChannel.Publish(
		broadCastExchange,   // exchange
		broadCastRoutingKey, // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(str),
			ReplyTo:     a.rpcQueue.Name,
		})
}
