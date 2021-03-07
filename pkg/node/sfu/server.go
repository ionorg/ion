package sfu

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/cloudwebrtc/nats-discovery/pkg/discovery"
	log "github.com/pion/ion-log"
	sfu "github.com/pion/ion-sfu/pkg/sfu"
	rtc "github.com/pion/ion/pkg/grpc/rtc"
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type sfuServer struct {
	rtc.UnimplementedRTCServer
	sync.Mutex
	SFU      *sfu.SFU
	nodeLock sync.RWMutex
	nodes    map[string]*discovery.Node
}

func newServer(sfu *sfu.SFU) *sfuServer {
	return &sfuServer{SFU: sfu, nodes: make(map[string]*discovery.Node)}
}

// watchNodes watch islb nodes up/down
func (s *sfuServer) watchNodes(state discovery.NodeState, node *discovery.Node) {
	s.nodeLock.Lock()
	defer s.nodeLock.Unlock()
	id := node.NID
	if state == discovery.NodeUp {
		log.Infof("islb node %v up", id)
		if _, found := s.nodes[id]; !found {
			s.nodes[id] = node
		}
	} else if state == discovery.NodeDown {
		log.Infof("islb node %v down", id)
		delete(s.nodes, id)
	}
}

func (s *sfuServer) Signal(stream rtc.RTC_SignalServer) error {
	peer := sfu.NewPeer(s.SFU)
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

			log.Errorf("signal error %v %v", errStatus.Message(), errStatus.Code())
			return err
		}

		switch payload := in.Payload.(type) {
		case *rtc.Signalling_Join:
			log.Debugf("signal->join sid:\n%v", string(payload.Join.GetReq().Sid))

			switch join := payload.Join.Payload.(type) {
			case *rtc.Join_Req:
				sid := join.Req.Sid
				uid := join.Req.Uid
				//parameters := join.Req.Parameters

				err := peer.Join(sid, uid)
				if err != nil {
					switch err {
					case sfu.ErrTransportExists:
						fallthrough
					case sfu.ErrOfferIgnored:
						err = stream.Send(&rtc.Signalling{
							Payload: &rtc.Signalling_Error{
								Error: &rtc.Error{
									Code:   500,
									Reason: fmt.Errorf("join error: %w", err).Error(),
								},
							},
						})
						if err != nil {
							log.Errorf("grpc send error %v ", err)
							return status.Errorf(codes.Internal, err.Error())
						}
					default:
						return status.Errorf(codes.Unknown, err.Error())
					}
				}

				stream.Send(&rtc.Signalling{
					Payload: &rtc.Signalling_Join{
						Join: &rtc.Join{
							Payload: &rtc.Join_Reply{
								Reply: &rtc.JoinReply{
									Success: true,
									Error:   "",
								},
							},
						},
					},
				})
			}

			/*
				var offer webrtc.SessionDescription
				err := json.Unmarshal(payload.Join.Description, &offer)
				if err != nil {
					s.Lock()
					err = stream.Send(&rtc.Signalling{
						Payload: &rtc.Signalling_Error{
							Error: fmt.Errorf("join sdp unmarshal error: %w", err).Error(),
						},
					})
					s.Unlock()
					if err != nil {
						log.Errorf("grpc send error %v ", err)
						return status.Errorf(codes.Internal, err.Error())
					}
				}
			*/
			// Notify user of new ice candidate
			peer.OnIceCandidate = func(candidate *webrtc.ICECandidateInit, target int) {
				bytes, err := json.Marshal(candidate)
				if err != nil {
					log.Errorf("OnIceCandidate error %s", err)
				}
				s.Lock()
				err = stream.Send(&rtc.Signalling{
					Payload: &rtc.Signalling_Trickle{
						Trickle: &rtc.Trickle{
							Candidate: bytes,
							Target:    rtc.Target(target),
						},
					},
				})
				s.Unlock()
				if err != nil {
					log.Errorf("OnIceCandidate send error %v ", err)
				}
			}

			// Notify user of new offer
			peer.OnOffer = func(o *webrtc.SessionDescription) {
				marshalled, err := json.Marshal(o)
				if err != nil {
					s.Lock()
					err = stream.Send(&rtc.Signalling{
						Payload: &rtc.Signalling_Error{
							Error: &rtc.Error{
								Code:   415,
								Reason: fmt.Errorf("offer sdp marshal error: %w", err).Error(),
							},
						},
					})
					s.Unlock()
					if err != nil {
						log.Errorf("grpc send error %v ", err)
					}
					return
				}

				s.Lock()
				err = stream.Send(&rtc.Signalling{
					Payload: &rtc.Signalling_Description{
						Description: &rtc.Description{
							Description: marshalled,
						},
					},
				})
				s.Unlock()

				if err != nil {
					log.Errorf("negotiation error %s", err)
				}
			}

			peer.OnICEConnectionStateChange = func(c webrtc.ICEConnectionState) {
				log.Infof("oniceconnectionstatechange: %v", c.String())
				if err != nil {
					log.Errorf("oniceconnectionstatechange error %s", err)
				}
			}
			/*
					err = peer.Join(payload.Join.Sid, webrtc.SessionDescription{})
					if err != nil {
						switch err {
						case sfu.ErrTransportExists:
							fallthrough
						case sfu.ErrOfferIgnored:
							s.Lock()
							err = stream.Send(&rtc.Signalling{
								Payload: &rtc.Signalling_Error{
									Error: fmt.Errorf("join error: %w", err).Error(),
								},
							})
							s.Unlock()
							if err != nil {
								log.Errorf("grpc send error %v ", err)
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
						s.Lock()
						err = stream.Send(&rtc.Signalling{
							Payload: &rtc.Signalling_Description{
								Description: &rtc.Description{
									Description: marshalled,
								},
							},
						})
						s.Unlock()

				if err != nil {
					log.Errorf("error sending join response %s", err)
					return status.Errorf(codes.Internal, "join error %s", err)
				}
			*/
		case *rtc.Signalling_Description:
			var sdp webrtc.SessionDescription
			err := json.Unmarshal(payload.Description.Description, &sdp)
			if err != nil {
				s.Lock()
				err = stream.Send(&rtc.Signalling{
					Payload: &rtc.Signalling_Error{
						Error: &rtc.Error{
							Code:   415,
							Reason: fmt.Errorf("negotiate sdp unmarshal error: %w", err).Error(),
						},
					},
				})
				s.Unlock()
				if err != nil {
					log.Errorf("grpc send error %v ", err)
					return status.Errorf(codes.Internal, err.Error())
				}
			}

			if sdp.Type == webrtc.SDPTypeOffer {
				answer, err := peer.Answer(sdp)
				if err != nil {
					switch err {
					case sfu.ErrNoTransportEstablished:
						fallthrough
					case sfu.ErrOfferIgnored:
						s.Lock()
						err = stream.Send(&rtc.Signalling{
							Payload: &rtc.Signalling_Error{
								Error: &rtc.Error{
									Code:   415,
									Reason: fmt.Errorf("negotiate answer error: %w", err).Error(),
								},
							},
						})
						s.Unlock()
						if err != nil {
							log.Errorf("grpc send error %v ", err)
							return status.Errorf(codes.Internal, err.Error())
						}
						continue
					default:
						return status.Errorf(codes.Unknown, fmt.Sprintf("negotiate error: %v", err))
					}
				}

				marshalled, err := json.Marshal(answer)
				if err != nil {
					s.Lock()
					err = stream.Send(&rtc.Signalling{
						Payload: &rtc.Signalling_Error{
							Error: &rtc.Error{
								Code:   415,
								Reason: fmt.Errorf("sdp marshal error: %w", err).Error(),
							},
						},
					})
					s.Unlock()
					if err != nil {
						log.Errorf("grpc send error %v ", err)
						return status.Errorf(codes.Internal, err.Error())
					}
				}

				s.Lock()
				err = stream.Send(&rtc.Signalling{
					Payload: &rtc.Signalling_Description{
						Description: &rtc.Description{
							Description: marshalled,
						},
					},
				})
				s.Unlock()

				if err != nil {
					return status.Errorf(codes.Internal, fmt.Sprintf("negotiate error: %v", err))
				}

			} else if sdp.Type == webrtc.SDPTypeAnswer {
				err := peer.SetRemoteDescription(sdp)
				if err != nil {
					switch err {
					case sfu.ErrNoTransportEstablished:
						s.Lock()
						err = stream.Send(&rtc.Signalling{
							Payload: &rtc.Signalling_Error{
								Error: &rtc.Error{
									Code:   415,
									Reason: fmt.Errorf("set remote description error: %w", err).Error(),
								},
							},
						})
						s.Unlock()
						if err != nil {
							log.Errorf("grpc send error %v ", err)
							return status.Errorf(codes.Internal, err.Error())
						}
					default:
						return status.Errorf(codes.Unknown, err.Error())
					}
				}
			}

		case *rtc.Signalling_Trickle:
			var candidate webrtc.ICECandidateInit
			err := json.Unmarshal([]byte(payload.Trickle.Candidate), &candidate)
			if err != nil {
				log.Errorf("error parsing ice candidate: %v", err)
				s.Lock()
				err = stream.Send(&rtc.Signalling{
					Payload: &rtc.Signalling_Error{
						Error: &rtc.Error{
							Code:   415,
							Reason: fmt.Errorf("unmarshal ice candidate error:  %w", err).Error(),
						},
					},
				})
				s.Unlock()
				if err != nil {
					log.Errorf("grpc send error %v ", err)
					return status.Errorf(codes.Internal, err.Error())
				}
				continue
			}

			err = peer.Trickle(candidate, int(payload.Trickle.Target))
			if err != nil {
				switch err {
				case sfu.ErrNoTransportEstablished:
					log.Errorf("peer hasn't joined")
					s.Lock()
					err = stream.Send(&rtc.Signalling{
						Payload: &rtc.Signalling_Error{
							Error: &rtc.Error{
								Code:   415,
								Reason: fmt.Errorf("trickle error:  %w", err).Error(),
							},
						},
					})
					s.Unlock()
					if err != nil {
						log.Errorf("grpc send error %v ", err)
						return status.Errorf(codes.Internal, err.Error())
					}
				default:
					return status.Errorf(codes.Unknown, fmt.Sprintf("negotiate error: %v", err))
				}
			}

		}
	}
}
