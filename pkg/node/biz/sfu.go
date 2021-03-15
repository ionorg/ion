package biz

import (
	"context"
	"fmt"
	"io"

	nrpc "github.com/cloudwebrtc/nats-grpc/pkg/rpc"
	log "github.com/pion/ion-log"
	sfu "github.com/pion/ion-sfu/cmd/signal/grpc/proto"
)

type SFUSignalBridge struct {
	sfu.UnimplementedSFUServer
	BizServer *BizServer
}

//Signal bridge SFU signaling between client and sfu node.
func (s *SFUSignalBridge) Signal(sstream sfu.SFU_SignalServer) error {
	var peer *Peer = nil
	var cstream sfu.SFU_SignalClient = nil
	reqCh := make(chan *sfu.SignalRequest)
	repCh := make(chan *sfu.SignalReply)
	errCh := make(chan error)

	defer func() {
		if cstream != nil {
			cstream.CloseSend()
		}
		close(errCh)
		log.Infof("SFU.Signal loop done")
	}()

	go func() {
		defer close(reqCh)
		for {
			req, err := sstream.Recv()
			if err != nil {
				log.Errorf("Singal server stream.Recv() err: %v", err)
				return
			}
			reqCh <- req
		}
	}()

	for {
		select {
		case err := <-errCh:
			return err
		case req, ok := <-reqCh:

			if !ok {
				return io.EOF
			}

			if cstream != nil {
				cstream.Send(req)
				break
			}

			switch payload := req.Payload.(type) {
			case *sfu.SignalRequest_Join:
				//TODO: Check if you have permission to connect to the SFU node
				r := s.BizServer.getRoom(payload.Join.Sid)
				if r != nil {
					peer = r.getPeer(payload.Join.Uid)
					if peer != nil {
						// Use nats-grpc or grpc
						// TODO: change to util.NewGRPCClientConnForNode.
						cli := sfu.NewSFUClient(nrpc.NewClient(s.BizServer.nc, r.sfunid))
						var err error
						cstream, err = cli.Signal(context.Background())
						if err != nil {
							log.Errorf("Singal cli.Signal() err: %v", err)
							return err
						}

						go func() {
							defer close(repCh)
							for {
								reply, err := cstream.Recv()
								if err != nil {
									log.Errorf("Singal client stream.Recv() err: %v", err)
									return
								}
								repCh <- reply
							}
						}()

						cstream.Send(req)
						break
					} else {
						return fmt.Errorf("peer [%v] not found", payload.Join.Uid)
					}
				} else {
					return fmt.Errorf("session [%v] not found", payload.Join.Sid)
				}
			}
		case reply, ok := <-repCh:
			if ok {
				sstream.Send(reply)
				break
			}
			return io.EOF
		}
	}
}
