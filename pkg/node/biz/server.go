package biz

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	biz "github.com/pion/ion/pkg/grpc/biz"
	islb "github.com/pion/ion/pkg/grpc/islb"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
)

// BizServer represents an BizServer instance
type BizServer struct {
	biz.UnimplementedBizServer
	nc       *nats.Conn
	elements []string
	roomLock sync.RWMutex
	rooms    map[string]*Room
	closed   chan bool
	islbcli  islb.ISLBClient
	bn       *BIZ
}

// newBizServer creates a new avp server instance
func newBizServer(bn *BIZ, c string, nid string, elements []string, nc *nats.Conn) *BizServer {
	return &BizServer{
		bn:       bn,
		nc:       nc,
		elements: elements,
		rooms:    make(map[string]*Room),
		closed:   make(chan bool),
	}
}

func (s *BizServer) close() {
	close(s.closed)
}

func (s *BizServer) createRoom(sid string, sfuNID string) *Room {
	s.roomLock.RLock()
	defer s.roomLock.RUnlock()
	r := newRoom(sid, sfuNID)
	s.rooms[sid] = r
	return r
}

func (s *BizServer) getRoom(id string) *Room {
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	r := s.rooms[id]
	return r
}

func (s *BizServer) delRoom(id string) {
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	delete(s.rooms, id)
}

func (s *BizServer) watchISLBEvent() {
	
}

//Signal process biz request.
func (s *BizServer) Signal(stream biz.Biz_SignalServer) error {
	var r *Room = nil
	var peer *Peer = nil
	errCh := make(chan error)
	repCh := make(chan *biz.SignalReply)
	reqCh := make(chan *biz.SignalRequest)

	defer func() {
		if peer != nil && r != nil {
			peer.Close()
			r.delPeer(peer.UID())
		}
		close(errCh)
		close(repCh)
		close(reqCh)
		log.Infof("BizServer.Signal loop done")
	}()

	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				log.Errorf("BizServer.Singal server stream.Recv() err: %v", err)
				errCh <- err
				return
			}
			reqCh <- req
		}
	}()

	for {
		select {
		case err := <-errCh:
			return err
		case reply, ok := <-repCh:
			if !ok {
				return io.EOF
			}
			stream.Send(reply)
		case req, ok := <-reqCh:
			if !ok {
				return io.EOF
			}
			log.Infof("Biz request => %v", req.String())

			switch payload := req.Payload.(type) {
			case *biz.SignalRequest_Join:
				sid := payload.Join.Peer.Sid
				uid := payload.Join.Peer.Uid

				success := false
				reason := "unkown error."

				if s.islbcli == nil {
					nodes := s.bn.GetNeighborNodes()
					for _, node := range nodes {
						if node.Service == proto.ServiceISLB {
							ncli := nrpc.NewClient(s.nc, node.NID)
							s.islbcli = islb.NewISLBClient(ncli)
							s.watchISLBEvent()
							break
						}
					}
				}

				if s.islbcli != nil {
					r = s.getRoom(sid)
					if r == nil {
						reason = fmt.Sprintf("room sid = %v not found", sid)
						resp, err := s.islbcli.FindNode(context.TODO(), &islb.FindNodeRequest{
							Service: proto.ServiceSFU,
							Sid:     sid,
						})

						if err == nil && len(resp.Nodes) > 0 {
							r = s.createRoom(sid, resp.GetNodes()[0].Nid)
						} else {
							reason = fmt.Sprintf("islbcli.FindNode(serivce = sfu, sid = %v) err %v", sid, err)
						}
					}
					if r != nil {
						peer = NewPeer(sid, uid, payload.Join.Peer.Info, repCh)
						r.addPeer(peer)
						success = true
						reason = "join success."
					}
				} else {
					reason = fmt.Sprintf("join [sid=%v] islb node not found", sid)
				}

				stream.Send(&biz.SignalReply{
					Payload: &biz.SignalReply_JoinReply{
						JoinReply: &biz.JoinReply{
							Success: success,
							Reason:  reason,
						},
					},
				})
			case *biz.SignalRequest_Leave:
				uid := payload.Leave.Uid
				if peer != nil && peer.uid == uid {
					r.delPeer(uid)
					peer.Close()
					peer = nil

					if r.count() == 0 {
						s.delRoom(r.SID())
						r = nil
					}

					stream.Send(&biz.SignalReply{
						Payload: &biz.SignalReply_LeaveReply{
							LeaveReply: &biz.LeaveReply{},
						},
					})
				}
			case *biz.SignalRequest_Msg:
				log.Debugf("Message: from: %v => to: %v, data: %v", payload.Msg.From, payload.Msg.To, payload.Msg.Data)
				// message broadcast
				r.sendMessage(payload.Msg)
			default:
				break
			}

		}
	}
}

// stat peers
func (s *BizServer) stat() {
	t := time.NewTicker(util.DefaultStatCycle)
	defer t.Stop()
	for {
		select {
		case <-t.C:
		case <-s.closed:
			log.Infof("stop stat")
			return
		}

		var info string
		s.roomLock.RLock()
		for sid, room := range s.rooms {
			info += fmt.Sprintf("room: %s\npeers: %d\n", sid, room.count())
		}
		s.roomLock.RUnlock()
		if len(info) > 0 {
			log.Infof("\n----------------signal-----------------\n" + info)
		}
	}
}
