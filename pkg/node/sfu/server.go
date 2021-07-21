package sfu

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	log "github.com/pion/ion-log"
	isfu "github.com/pion/ion-sfu/pkg/sfu"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/ion/proto/ion"
	"github.com/pion/ion/proto/islb"
	pb "github.com/pion/ion/proto/sfu"
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type sfuServer struct {
	pb.UnimplementedSFUServer
	sfu     *isfu.SFU
	islbcli islb.ISLBClient
	sn      *SFU
}

func newSFUServer(sn *SFU, sfu *isfu.SFU) *sfuServer {
	return &sfuServer{sn: sn, sfu: sfu}
}

func (s *sfuServer) postISLBEvent(event *islb.ISLBEvent) {

	if s.islbcli == nil {
		ncli, err := s.sn.NewNatsRPCClient(proto.ServiceISLB, "*", map[string]interface{}{})
		if err != nil {
			log.Errorf("NewNatsRPCClient err %v", err)
			return
		}
		s.islbcli = islb.NewISLBClient(ncli)
	}

	if s.islbcli != nil {
		_, err := s.islbcli.PostISLBEvent(context.Background(), event)
		if err != nil {
			log.Errorf("PostISLBEvent err %v", err)
		}
	}
}

func (s *sfuServer) Signal(stream pb.SFU_SignalServer) error {
	recvCandidates := []webrtc.ICECandidateInit{}
	peer := isfu.NewPeer(s.sfu)
	var streams []*ion.Stream

	defer func() {
		if peer.Session() != nil {
			s.postISLBEvent(&islb.ISLBEvent{
				Payload: &islb.ISLBEvent_Stream{
					Stream: &ion.StreamEvent{
						Nid:     s.sn.NID,
						Sid:     peer.Session().ID(),
						Uid:     peer.ID(),
						State:   ion.StreamEvent_REMOVE,
						Streams: streams,
					},
				},
			})
		}
	}()

	for {
		in, err := stream.Recv()

		if err != nil {
			peer.Close()

			if err == io.EOF {
				return nil
			}

			errStatus, _ := status.FromError(err)
			if errStatus.Code() == codes.Canceled {
				return nil
			}

			log.Errorf("%v signal error %d", fmt.Errorf(errStatus.Message()), errStatus.Code())
			return err
		}

		switch payload := in.Payload.(type) {
		case *pb.SignalRequest_Join:
			log.Infof("signal->join called => %v", string(payload.Join.Description))

			var offer webrtc.SessionDescription
			err := json.Unmarshal(payload.Join.Description, &offer)
			if err != nil {
				err = stream.Send(&pb.SignalReply{
					Payload: &pb.SignalReply_Error{
						Error: fmt.Errorf("join sdp unmarshal error: %w", err).Error(),
					},
				})
				if err != nil {
					log.Errorf("grpc send error: %v", err)
					return status.Errorf(codes.Internal, err.Error())
				}
			}

			// Notify user of new ice candidate
			peer.OnIceCandidate = func(candidate *webrtc.ICECandidateInit, target int) {
				bytes, err := json.Marshal(candidate)
				if err != nil {
					log.Errorf("OnIceCandidate error: %v", err)
				}
				err = stream.Send(&pb.SignalReply{
					Payload: &pb.SignalReply_Trickle{
						Trickle: &pb.Trickle{
							Init:   string(bytes),
							Target: pb.Trickle_Target(target),
						},
					},
				})
				if err != nil {
					log.Errorf("OnIceCandidate send error: %v", err)
				}
			}

			// Notify user of new offer
			peer.OnOffer = func(o *webrtc.SessionDescription) {
				marshalled, err := json.Marshal(o)
				if err != nil {
					err = stream.Send(&pb.SignalReply{
						Payload: &pb.SignalReply_Error{
							Error: fmt.Errorf("offer sdp marshal error: %w", err).Error(),
						},
					})
					if err != nil {
						log.Errorf("grpc send error: %v", err)
					}
					return
				}

				err = stream.Send(&pb.SignalReply{
					Payload: &pb.SignalReply_Description{
						Description: marshalled,
					},
				})

				if err != nil {
					log.Errorf("negotiation error: %v", err)
				}
			}

			peer.OnICEConnectionStateChange = func(c webrtc.ICEConnectionState) {
				err = stream.Send(&pb.SignalReply{
					Payload: &pb.SignalReply_IceConnectionState{
						IceConnectionState: c.String(),
					},
				})

				if err != nil {
					log.Errorf("oniceconnectionstatechange error: %v", err)
				}
			}

			err = peer.Join(payload.Join.Sid, payload.Join.Uid)
			if err != nil {
				switch err {
				case isfu.ErrTransportExists:
					fallthrough
				case isfu.ErrOfferIgnored:
					err = stream.Send(&pb.SignalReply{
						Payload: &pb.SignalReply_Error{
							Error: fmt.Errorf("join error: %w", err).Error(),
						},
					})
					if err != nil {
						log.Errorf("grpc send error: %v", err)
						return status.Errorf(codes.Internal, err.Error())
					}
				default:
					return status.Errorf(codes.Unknown, err.Error())
				}
			}

			answer, err := peer.Answer(offer)
			if err != nil {
				return status.Errorf(codes.Internal, fmt.Sprintf("answer error: %v", err))
			}

			marshalled, err := json.Marshal(answer)
			if err != nil {
				return status.Errorf(codes.Internal, fmt.Sprintf("sdp marshal error: %v", err))
			}

			// send answer
			err = stream.Send(&pb.SignalReply{
				Id: in.Id,
				Payload: &pb.SignalReply_Join{
					Join: &pb.JoinReply{
						Description: marshalled,
					},
				},
			})

			if err != nil {
				log.Errorf("error sending join response, error -> %v", err)
				return status.Errorf(codes.Internal, "join error %s", err)
			}

		case *pb.SignalRequest_Description:
			var sdp webrtc.SessionDescription
			err := json.Unmarshal(payload.Description, &sdp)
			if err != nil {
				err = stream.Send(&pb.SignalReply{
					Payload: &pb.SignalReply_Error{
						Error: fmt.Errorf("negotiate sdp unmarshal error: %w", err).Error(),
					},
				})
				if err != nil {
					log.Errorf("grpc send error: %v", err)
					return status.Errorf(codes.Internal, err.Error())
				}
			}

			if sdp.Type == webrtc.SDPTypeOffer {
				answer, err := peer.Answer(sdp)
				if err != nil {
					switch err {
					case isfu.ErrNoTransportEstablished:
						fallthrough
					case isfu.ErrOfferIgnored:
						err = stream.Send(&pb.SignalReply{
							Payload: &pb.SignalReply_Error{
								Error: fmt.Errorf("negotiate answer error: %w", err).Error(),
							},
						})
						if err != nil {
							log.Errorf("grpc send error: %v", err)
							return status.Errorf(codes.Internal, err.Error())
						}
						continue
					default:
						return status.Errorf(codes.Unknown, fmt.Sprintf("negotiate error: %v", err))
					}
				}

				marshalled, err := json.Marshal(answer)
				if err != nil {
					err = stream.Send(&pb.SignalReply{
						Payload: &pb.SignalReply_Error{
							Error: fmt.Errorf("sdp marshal error: %w", err).Error(),
						},
					})
					if err != nil {
						log.Errorf("grpc send error: %v", err)
						return status.Errorf(codes.Internal, err.Error())
					}
				}
				err = stream.Send(&pb.SignalReply{
					Id: in.Id,
					Payload: &pb.SignalReply_Description{
						Description: marshalled,
					},
				})

				if err != nil {
					return status.Errorf(codes.Internal, fmt.Sprintf("negotiate error: %v", err))
				}

				newStreams, err := util.ParseSDP(sdp.SDP)
				if err != nil {
					log.Errorf("util.ParseSDP error: %v", err)
				}

				if len(newStreams) > 0 {
					s.postISLBEvent(&islb.ISLBEvent{
						Payload: &islb.ISLBEvent_Stream{
							Stream: &ion.StreamEvent{
								Nid:     s.sn.NID,
								Sid:     peer.Session().ID(),
								Uid:     peer.ID(),
								Streams: newStreams,
								State:   ion.StreamEvent_ADD,
							},
						},
					})

					streams = newStreams
				}

			} else if sdp.Type == webrtc.SDPTypeAnswer {
				log.Infof("sdp %v", sdp.SDP)
				err := peer.SetRemoteDescription(sdp)
				if err != nil {
					switch err {
					case isfu.ErrNoTransportEstablished:
						err = stream.Send(&pb.SignalReply{
							Payload: &pb.SignalReply_Error{
								Error: fmt.Errorf("set remote description error: %w", err).Error(),
							},
						})
						if err != nil {
							log.Errorf("grpc send error: %v", err)
							return status.Errorf(codes.Internal, err.Error())
						}
					default:
						return status.Errorf(codes.Unknown, err.Error())
					}
				}
			}

		case *pb.SignalRequest_Trickle:
			var candidate webrtc.ICECandidateInit
			err := json.Unmarshal([]byte(payload.Trickle.Init), &candidate)
			if err != nil {
				log.Errorf("error parsing ice candidate, error -> %v", err)
				err = stream.Send(&pb.SignalReply{
					Payload: &pb.SignalReply_Error{
						Error: fmt.Errorf("unmarshal ice candidate error:  %w", err).Error(),
					},
				})
				if err != nil {
					log.Errorf("grpc send error: %v", err)
					return status.Errorf(codes.Internal, err.Error())
				}
				continue
			}

			// trickle the old candidates
			for _, candidate := range recvCandidates {
				err = peer.Trickle(candidate, int(payload.Trickle.Target))
				if err != nil {
					log.Warnf("peer trickle error: %v", err)
				}
			}
			err = peer.Trickle(candidate, int(payload.Trickle.Target))
			if err != nil {
				switch err {
				case isfu.ErrNoTransportEstablished:
					// cadidate arrived before join, cache it
					log.Infof("cadidate arrived before join, cache it: %v", candidate)
					recvCandidates = append(recvCandidates, candidate)
				default:
					return status.Errorf(codes.Unknown, fmt.Sprintf("negotiate error: %v", err))
				}
			}
		}
	}
}
