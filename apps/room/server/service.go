package server

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	log "github.com/pion/ion-log"
	biz_pb "github.com/pion/ion/apps/biz/proto"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/ion/proto/ion"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type SFUInteractive interface {
	WatchStreamEvent(sid string) (chan *ion.StreamEvent, context.CancelFunc, error)
	GetNode(service, sid, uid string) (string, error)
}

// BizService represents an BizService instance
type BizService struct {
	biz_pb.UnimplementedBizServer
	roomLock sync.RWMutex
	rooms    map[string]*Room
	closed   chan struct{}
	sfu      SFUInteractive
}

// NewBizService creates a new avp server instance
func NewBizService(sfu SFUInteractive) (*BizService, error) {
	b := &BizService{
		rooms:  make(map[string]*Room),
		closed: make(chan struct{}),
		sfu:    sfu,
	}
	return b, nil
}

func (b *BizService) RegisterService(registrar grpc.ServiceRegistrar) {
	biz_pb.RegisterBizServer(registrar, b)
}

func (s *BizService) Close() {
	close(s.closed)
}

func (s *BizService) createRoom(sid string, sfuNID string) *Room {
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	if r := s.rooms[sid]; r != nil {
		return r
	}
	r := newRoom(sid, sfuNID)
	s.rooms[sid] = r
	return r
}

func (s *BizService) getRoom(id string) *Room {
	s.roomLock.RLock()
	defer s.roomLock.RUnlock()
	return s.rooms[id]
}

func (s *BizService) delRoom(r *Room) {
	id := r.SID()
	s.roomLock.Lock()
	defer s.roomLock.Unlock()
	if s.rooms[id] == r {
		delete(s.rooms, id)
	}
}

func (s *BizService) watchStreamEvent(sid string) (context.CancelFunc, error) {
	ch, cancel, err := s.sfu.WatchStreamEvent(sid)
	if err != nil {
		log.Errorf("s.watchStreamEvent failed %v", err)
		return nil, err
	}

	go func() {
		for event := range ch {
			log.Infof("stream event %v", event)
			r := s.getRoom(event.Sid)
			if r != nil {
				r.sendStreamEvent(event)
				p := r.getPeer(event.Uid)
				// save last stream info.
				if p != nil {
					p.lastStreamEvent = event
				}
			}
			//TODO: add context for stop
		}
	}()
	return cancel, nil
}

//Signal process biz request.
func (s *BizService) Signal(stream biz_pb.Biz_SignalServer) error {
	var cancel context.CancelFunc = nil
	var r *Room = nil
	var peer *Peer = nil
	errCh := make(chan error)
	repCh := make(chan *biz_pb.SignalReply, 1)
	reqCh := make(chan *biz_pb.SignalRequest)

	defer func() {
		if r != nil {
			if peer != nil {
				peer.Close()
				r.delPeer(peer)
			}
			if r.count() == 0 {
				s.delRoom(r)
				if cancel != nil {
					cancel()
				}
			}
		}
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
			case *biz_pb.SignalRequest_Join:
				sid := payload.Join.Peer.Sid
				uid := payload.Join.Peer.Uid

				success := false
				reason := "unkown error."
				r = s.getRoom(sid)

				if r == nil {
					reason = fmt.Sprintf("room sid = %v not found", sid)
					nid, err := s.sfu.GetNode(proto.ServiceRTC, sid, uid)
					if err != nil {
						log.Errorf("dnc.Get: serivce = %v error %v", proto.ServiceRTC, err)
						reason = "get serivce [sfu], node cnt == 0"
					} else {
						r = s.createRoom(sid, nid)
						cancel, err = s.watchStreamEvent(sid)
						if err != nil {
							log.Warnf("failed to watch stream event %v", err)
						}
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

				err := stream.Send(&biz_pb.SignalReply{
					Payload: &biz_pb.SignalReply_JoinReply{
						JoinReply: &biz_pb.JoinReply{
							Success: success,
							Reason:  reason,
						},
					},
				})

				if err != nil {
					log.Errorf("stream.Send(&biz_pb.SignalReply) failed %v", err)
				}
			case *biz_pb.SignalRequest_Leave:
				uid := payload.Leave.Uid
				if peer != nil && peer.uid == uid {
					if r.delPeer(peer) == 0 {
						s.delRoom(r)
						r = nil
					}
					peer.Close()
					peer = nil

					err := stream.Send(&biz_pb.SignalReply{
						Payload: &biz_pb.SignalReply_LeaveReply{
							LeaveReply: &biz_pb.LeaveReply{
								Reason: "closed",
							},
						},
					})
					if err != nil {
						log.Errorf("stream.Send(&biz_pb.SignalReply) failed %v", err)
					}
				}
			case *biz_pb.SignalRequest_Msg:
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
func (s *BizService) stat() {
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
