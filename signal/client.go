package signal

import (
	"crypto/tls"
	"errors"

	"sync"

	"github.com/centrifugal/centrifuge-go"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/pion/sfu/conf"
	"github.com/pion/sfu/log"
)

type EventCallback func(event, channel, data string)

var (
	ErrorInvalidParams = errors.New("invalid params")
)

const (
	EventOnConnect          = "OnConnect"
	EventOnDisconnect       = "OnDisconnect"
	EventOnPublish          = "OnPublish"
	EventOnPrivateSub       = "OnPrivateSub"
	EventOnError            = "OnError"
	EventOnSubscribeSuccess = "OnSubscribeSuccess"
	EventOnSubscribeError   = "OnSubscribeError"
	EventOnUnsubscribe      = "OnUnsubscribe"
)

func NewClient() *Client {
	cfg := conf.Cfg.Centrifugo
	if cfg.Url == "" || cfg.CertPem == "" || cfg.KeyPem == "" || cfg.Key == "" {
		log.Panicf(ErrorInvalidParams.Error())
		return nil
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertPem, cfg.KeyPem)
	if err != nil {
		log.Panicf(err.Error())
		return nil
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
	}
	tlsConfig.BuildNameToCertificate()

	c := &Client{
		client: centrifuge.New(cfg.Url, centrifuge.Config{
			PingInterval:     centrifuge.DefaultPingInterval,
			ReadTimeout:      centrifuge.DefaultReadTimeout,
			WriteTimeout:     centrifuge.DefaultWriteTimeout,
			HandshakeTimeout: centrifuge.DefaultHandshakeTimeout,
			TLSConfig:        tlsConfig,
		}),
		clientLock: new(sync.RWMutex),
	}

	c.client.SetToken(c.createConnectToken("sfu"))
	c.client.OnPrivateSub(c)
	c.client.OnDisconnect(c)
	c.client.OnConnect(c)
	c.client.OnError(c)

	return c
}

type Client struct {
	client     *centrifuge.Client
	clientLock *sync.RWMutex
	Handler
	eventCall EventCallback
	clientID  string
}

func (c *Client) createConnectToken(room string) string {
	claims := jwt.MapClaims{"sub": room}
	if conf.Cfg.Centrifugo.Expire > 0 {
		claims["exp"] = conf.Cfg.Centrifugo.Expire
	}
	t, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(conf.Cfg.Centrifugo.Key))
	if err != nil {
		log.Errorf(err.Error())
	}
	return t
}

func (c *Client) Subscribe(channel string) *centrifuge.Subscription {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	sub, err := c.client.NewSubscription(channel)
	if err != nil {
		log.Errorf(err.Error())
		return nil
	}
	sub.OnSubscribeSuccess(c)
	sub.OnSubscribeError(c)
	sub.OnUnsubscribe(c)
	sub.OnPublish(c)

	if err := sub.Subscribe(); err != nil {
		log.Errorf(err.Error())
	}
	log.Infof("Client.Subscribe %s", channel)
	return sub
}

func (c *Client) Publish(channel, data string) error {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	err := c.client.Publish(channel, []byte(data))
	log.Infof("Client.Publish %s %s", channel, data)
	return err
}

func (c *Client) SetEventCallback(eventCall EventCallback) error {
	log.Infof("Client.SetEventCallback %v", eventCall)
	c.eventCall = eventCall
	return nil
}

func (c *Client) Close() error {
	log.Infof("Client.Close")
	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	err := c.client.Close()
	return err
}

func (c *Client) Connect() error {
	c.clientLock.Lock()
	defer c.clientLock.Unlock()
	err := c.client.Connect()
	return err
}

func (c *Client) OnSubscribeSuccess(sub *centrifuge.Subscription, e centrifuge.SubscribeSuccessEvent) {
	log.Infof("OnSubscribeSuccess channel=%s", sub.Channel())
	// s.eventCall(EventOnSubscribeSuccess, sub.Channel(), "")
}

func (c *Client) OnPublish(sub *centrifuge.Subscription, e centrifuge.PublishEvent) {
	//skip self message
	if c.clientID != e.GetInfo().GetClient() {
		c.eventCall(EventOnPublish, sub.Channel(), string(e.Data))
	}
}

func (c *Client) OnConnect(cc *centrifuge.Client, e centrifuge.ConnectEvent) {
	c.clientID = e.ClientID
	// c.eventCall(EventOnConnect, "", e.ClientID)
}
