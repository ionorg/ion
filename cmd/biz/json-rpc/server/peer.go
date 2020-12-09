package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gorilla/websocket"
	log "github.com/pion/ion-log"
	"github.com/pion/ion/pkg/node/biz"
	"github.com/pion/ion/pkg/proto"
	"github.com/sourcegraph/jsonrpc2"
	websocketjsonrpc2 "github.com/sourcegraph/jsonrpc2/websocket"
)

// Peer represents a peer for client
type Peer struct {
	*biz.Peer
	auth func(proto.Authenticatable) error
	ctx  context.Context
	conn *jsonrpc2.Conn
}

// newPeer create a peer instance
func newPeer(ctx context.Context, c *websocket.Conn, uid proto.UID, bs *biz.Server, auth func(proto.Authenticatable) error) *Peer {
	p := &Peer{
		auth: auth,
		ctx:  ctx,
	}
	p.conn = jsonrpc2.NewConn(ctx, websocketjsonrpc2.NewObjectStream(c), p)
	p.Peer = biz.NewPeer(uid, bs, p.send)
	return p
}

// Handle incoming RPC call events, implement jsonrpc2.Handler
func (p *Peer) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	replyError := func(err error) {
		_ = conn.ReplyWithError(ctx, req.ID, &jsonrpc2.Error{
			Code:    500,
			Message: fmt.Sprintf("%s", err),
		})
	}

	if req.Params == nil {
		log.Errorf("request without params: uid=%s", p.UID())
		replyError(errors.New("request without params"))
		return
	}

	switch req.Method {
	case proto.ClientJoin:
		var msg proto.FromClientJoinMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing offer: %v", err)
			replyError(err)
			break
		}
		answer, err := p.Join(&msg)
		if err != nil {
			replyError(err)
			break
		}
		_ = conn.Reply(ctx, req.ID, answer)

	case proto.ClientOffer:
		var msg proto.ClientOfferMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing trickle: %v", err)
			replyError(err)
		}
		answer, err := p.Offer(&msg)
		if err != nil {
			replyError(err)
			break
		}
		_ = conn.Reply(ctx, req.ID, answer)

	case proto.ClientAnswer:
		var msg proto.ClientAnswerMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing trickle: %v", err)
			replyError(err)
			break
		}
		if err := p.Answer(&msg); err != nil {
			replyError(err)
		}

	case proto.ClientTrickle:
		var msg proto.ClientTrickleMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing trickle: %v", err)
			replyError(err)
			break
		}
		if err := p.Trickle(&msg); err != nil {
			replyError(err)
		}

	case proto.ClientLeave:
		var msg proto.FromClientLeaveMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing leave: %v", err)
			replyError(err)
			break
		}
		if err := p.Leave(&msg); err != nil {
			replyError(err)
		}

	case proto.ClientBroadcast:
		var msg proto.FromClientBroadcastMsg
		if err := p.unmarshal(*req.Params, &msg); err != nil {
			log.Errorf("error parsing trickle: %v", err)
			replyError(err)
			break
		}
		if err := p.Broadcast(&msg); err != nil {
			replyError(err)
		}

	default:
		replyError(errors.New("unknown message"))
	}
}

// unmarshal message
func (p *Peer) unmarshal(data json.RawMessage, result interface{}) error {
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	if p.auth != nil {
		if msg, ok := result.(proto.Authenticatable); ok {
			return p.auth(msg)
		}
	}
	return nil
}

// send message to the peer
func (p *Peer) send(msg interface{}) error {
	switch v := msg.(type) {
	case *proto.ClientOfferMsg:
		return p.conn.Notify(p.ctx, proto.ClientOffer, v.Desc)
	case *proto.ClientTrickleMsg:
		return p.conn.Notify(p.ctx, proto.ClientTrickle, v)
	case *proto.ToClientPeersMsg:
		return p.conn.Notify(p.ctx, proto.ClientOnList, v)
	case *proto.FromIslbStreamAddMsg:
		return p.conn.Notify(p.ctx, proto.ClientOnStreamAdd, v)
	case *proto.ToClientPeerJoinMsg:
		return p.conn.Notify(p.ctx, proto.ClientOnJoin, v)
	case *proto.IslbPeerLeaveMsg:
		return p.conn.Notify(p.ctx, proto.ClientOnLeave, v)
	case *proto.IslbBroadcastMsg:
		return p.conn.Notify(p.ctx, proto.ClientBroadcast, v)
	}
	return errors.New("unknown message")
}
