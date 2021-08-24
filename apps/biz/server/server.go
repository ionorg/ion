package server

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	ndc "github.com/cloudwebrtc/nats-discovery/pkg/client"
	"github.com/nats-io/nats.go"
	log "github.com/pion/ion-log"
	biz "github.com/pion/ion/apps/biz/proto"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	islb "github.com/pion/ion/proto/islb"
	"google.golang.org/grpc/metadata"
)

// BizServer represents an BizServer instance
type BizServer struct {
	biz.UnimplementedBizServer
	nc       *nats.Conn
	roomLock sync.RWMutex
	rooms    map[string]*Room
	closed   chan struct{}
	ndc      *ndc.Client
	islbcli  islb.ISLBClient
	bn       *BIZ
	stream   islb.ISLB_WatchISLBEventClient
}

// newBizServer creates a new avp server instance
func newBizServer(bn *BIZ, c string, nid string, nc *nats.Conn) (*BizServer, error) {

	ndc, err := ndc.NewClient(nc)
	if err != nil {
		log.Errorf("failed to create discovery client: %v", err)
		ndc.Close()
		return nil, err
	}

	b := &BizServer{
		ndc:    ndc,
		bn:     bn,
		nc:     nc,
		rooms:  make(map[string]*Room),
		closed: make(chan struct{}),
		stream: nil,
	}

	return b, nil
}

func (s *BizServer) close() {
	close(s.closed)
}

func (s *BizServer) createRoom(sid string, sfuNID string) *Room {
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	if r := s.rooms[sid]; r != nil {
		return r
	}
	r := newRoom(sid, sfuNID)
	s.rooms[sid] = r
	return r
}

func (s *BizServer) getRoom(id string) *Room {
	s.roomLock.RLock()
	defer s.roomLock.RUnlock()
	return s.rooms[id]
}

func (s *BizServer) delRoom(r *Room) {
	id := r.SID()
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	if s.rooms[id] == r {
		delete(s.rooms, id)
	}
}

func (s *BizServer) watchISLBEvent(nid string, sid string) error {
	//s.islbLock.Lock()
	//defer s.islbLock.Unlock()

	if s.islbcli == nil {
		ncli, err := s.bn.NewNatsRPCClient(proto.ServiceISLB, nid, map[string]interface{}{})
		if err != nil {
			return err
		}
		s.islbcli = islb.NewISLBClient(ncli)
	}

	if s.stream == nil && s.islbcli != nil {
		stream, err := s.islbcli.WatchISLBEvent(context.Background())
		if err != nil {
			return err
		}
		err = stream.Send(&islb.WatchRequest{
			Nid: nid,
			Sid: sid,
		})
		if err != nil {
			return err
		}

		go func() {
			defer func() {
				//s.islbLock.Lock()
				//defer s.islbLock.Unlock()
				s.stream = nil
			}()

			for {
				req, err := stream.Recv()
				if err != nil {
					log.Errorf("BizServer.Signal server stream.Recv() err: %v", err)
					return
				}
				log.Infof("watchISLBEvent req => %v", req)
				switch payload := req.Payload.(type) {
				case *islb.ISLBEvent_Stream:
					r := s.getRoom(payload.Stream.Sid)
					if r != nil {
						r.sendStreamEvent(payload.Stream)
						p := r.getPeer(payload.Stream.Uid)
						// save last stream info.
						if p != nil {
							p.lastStreamEvent = payload.Stream
						}
					}
				}
			}
		}()

		s.stream = stream
	}
	return nil
}

//Signal process biz request.
func (s *BizServer) Signal(stream biz.Biz_SignalServer) error {
	var r *Room = nil
	var peer *Peer = nil
	errCh := make(chan error)
	repCh := make(chan *biz.SignalReply, 1)
	reqCh := make(chan *biz.SignalRequest)

	defer func() {
		if r != nil {
			if peer != nil {
				peer.Close()
				r.delPeer(peer)
			}
			if r.count() == 0 {
				s.delRoom(r)
			}
		}

		log.Infof("BizServer.Signal loop done")
	}()

	go func() {
		for {
			req, err := stream.Recv()
			if err != nil {
				log.Errorf("BizServer.Signal server stream.Recv() err: %v", err)
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
			err := stream.Send(reply)
			if err != nil {
				return err
			}
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
				reason := "unknown error."
				r = s.getRoom(sid)

				if r == nil {
					reason = fmt.Sprintf("room sid = %v not found", sid)
					resp, err := s.ndc.Get(proto.ServiceSFU, map[string]interface{}{"sid": sid, "uid": uid})
					if err != nil {
						log.Errorf("dnc.Get: service = %v error %v", proto.ServiceSFU, err)
					}
					nid := ""
					if err == nil && len(resp.Nodes) > 0 {
						nid = resp.Nodes[0].NID
						r = s.createRoom(sid, nid)
						err = s.watchISLBEvent(nid, sid)
						if err != nil {
							log.Errorf("s.watchISLBEvent(req) failed %v", err)
						}
					} else {
						reason = "get service [sfu], node cnt == 0"
					}
				}

				if r != nil {
					peer = NewPeer(sid, uid, payload.Join.Peer.Info, repCh)
					r.addPeer(peer)
					success = true
					reason = "join success."

					//Generate necessary metadata for routing.
					header := metadata.New(map[string]string{"service": "sfu", "nid": r.nid, "sid": sid, "uid": uid})
					err := stream.SendHeader(header)
					if err != nil {
						log.Errorf("stream.SendHeader failed %v", err)
					}
				}

				err := stream.Send(&biz.SignalReply{
					Payload: &biz.SignalReply_JoinReply{
						JoinReply: &biz.JoinReply{
							Success: success,
							Reason:  reason,
						},
					},
				})

				if err != nil {
					log.Errorf("stream.Send(&biz.SignalReply) failed %v", err)
				}
			case *biz.SignalRequest_Leave:
				uid := payload.Leave.Uid
				if peer != nil && peer.uid == uid {
					if r.delPeer(peer) == 0 {
						s.delRoom(r)
						r = nil
					}
					peer.Close()
					peer = nil

					err := stream.Send(&biz.SignalReply{
						Payload: &biz.SignalReply_LeaveReply{
							LeaveReply: &biz.LeaveReply{
								Reason: "closed",
							},
						},
					})
					if err != nil {
						log.Errorf("stream.Send(&biz.SignalReply) failed %v", err)
					}
				}
			case *biz.SignalRequest_Msg:
				log.Debugf("Message: from: %v => to: %v, data: %v", payload.Msg.From, payload.Msg.To, payload.Msg.Data)
				// message broadcast
				if r != nil {
					r.sendMessage(payload.Msg)
				} else {
					log.Warnf("room not found, maybe the peer did not join")
				}
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
