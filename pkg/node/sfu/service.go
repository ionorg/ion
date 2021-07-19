package sfu

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	log "github.com/pion/ion-log"
	"github.com/pion/ion-sfu/pkg/middlewares/datachannel"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	error_code "github.com/pion/ion/pkg/error"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type SFUService struct {
	rtc.UnimplementedRTCServer
	sfu   *ion_sfu.SFU
	mutex sync.RWMutex
	sigs  map[string]rtc.RTC_SignalServer
}

func NewSFUService(conf ion_sfu.Config) *SFUService {
	s := &SFUService{
		sigs: make(map[string]rtc.RTC_SignalServer),
	}
	sfu := ion_sfu.NewSFU(conf)
	dc := sfu.NewDatachannel(ion_sfu.APIChannelLabel)
	dc.Use(datachannel.SubscriberAPI)
	s.sfu = sfu
	return s
}

func (s *SFUService) RegisterService(registrar grpc.ServiceRegistrar) {
	rtc.RegisterRTCServer(registrar, s)
}

func (s *SFUService) Close() {
	log.Infof("SFU service closed")
}

func (s *SFUService) BroadcastStreamEvent(event *rtc.StreamEvent) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for _, sig := range s.sigs {
		sig.Send(&rtc.Signalling{
			Payload: &rtc.Signalling_StreamEvent{
				StreamEvent: event,
			},
		})
	}
}

func (s *SFUService) Signal(stream rtc.RTC_SignalServer) error {
	peer := ion_sfu.NewPeer(s.sfu)
	var streams []*rtc.Stream

	defer func() {
		if peer.Session() != nil {
			event := &rtc.StreamEvent{
				State:   rtc.StreamEvent_REMOVE,
				Streams: streams,
			}
			s.BroadcastStreamEvent(event)
			log.Infof("broadcast stream event %v, state = REMOVE", event)
		}
		s.mutex.Lock()
		defer s.mutex.Unlock()
		delete(s.sigs, peer.ID())
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
		case *rtc.Signalling_Join:
			log.Infof("join: sid => %v, uid => %v", payload.Join.Sid, payload.Join.Uid)

			// Notify user of new ice candidate
			peer.OnIceCandidate = func(candidate *webrtc.ICECandidateInit, target int) {
				bytes, err := json.Marshal(candidate)
				if err != nil {
					log.Errorf("OnIceCandidate error: %v", err)
				}
				err = stream.Send(&rtc.Signalling{
					Payload: &rtc.Signalling_Trickle{
						Trickle: &rtc.Trickle{
							Init:   bytes,
							Target: rtc.Target(target),
						},
					},
				})
				if err != nil {
					log.Errorf("OnIceCandidate send error: %v", err)
				}
			}

			// Notify user of new offer
			peer.OnOffer = func(o *webrtc.SessionDescription) {
				err = stream.Send(&rtc.Signalling{
					Payload: &rtc.Signalling_Description{
						Description: &rtc.SessionDescription{
							Target: rtc.Target(rtc.Target_SUBSCRIBER),
							Sdp:    o.SDP,
							Type:   o.Type.String(),
						},
					},
				})
				if err != nil {
					log.Errorf("negotiation error: %v", err)
				}
			}

			err = peer.Join(payload.Join.Sid, payload.Join.Uid)
			if err != nil {
				switch err {
				case ion_sfu.ErrTransportExists:
					fallthrough
				case ion_sfu.ErrOfferIgnored:
					err = stream.Send(&rtc.Signalling{
						Payload: &rtc.Signalling_Error{
							Error: &rtc.Error{
								Code:   int32(error_code.InternalError),
								Reason: fmt.Sprintf("join error: %v", err),
							},
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

			s.mutex.Lock()
			defer s.mutex.Unlock()
			s.sigs[peer.ID()] = stream

		case *rtc.Signalling_Description:

			desc := webrtc.SessionDescription{
				SDP:  payload.Description.Sdp,
				Type: webrtc.NewSDPType(payload.Description.Type),
			}
			var err error = nil
			switch desc.Type {
			case webrtc.SDPTypeOffer:
				log.Infof("description: offer %v", desc.SDP)
				answer, err := peer.Answer(desc)
				if err != nil {
					return status.Errorf(codes.Internal, fmt.Sprintf("answer error: %v", err))
				}

				// send answer
				err = stream.Send(&rtc.Signalling{
					Payload: &rtc.Signalling_Description{
						Description: &rtc.SessionDescription{
							Target: rtc.Target(rtc.Target_PUBLISHER),
							Sdp:    answer.SDP,
							Type:   answer.Type.String(),
						},
					},
				})

				if err != nil {
					log.Errorf("grpc send error: %v", err)
					return status.Errorf(codes.Internal, err.Error())
				}

				newStreams, err := ParseSDP(peer.ID(), desc.SDP)
				if err != nil {
					log.Errorf("util.ParseSDP error: %v", err)
				}

				if len(newStreams) > 0 {
					event := &rtc.StreamEvent{
						Streams: newStreams,
						State:   rtc.StreamEvent_ADD,
					}
					streams = newStreams
					log.Infof("broadcast stream event %v, state = ADD", event)
					s.BroadcastStreamEvent(event)
				}

			case webrtc.SDPTypeAnswer:
				log.Infof("description: answer %v", desc.SDP)
				err = peer.SetRemoteDescription(desc)
			}

			if err != nil {
				switch err {
				case ion_sfu.ErrNoTransportEstablished:
					err = stream.Send(&rtc.Signalling{
						Payload: &rtc.Signalling_Error{
							Error: &rtc.Error{
								Code:   int32(error_code.UnsupportedMediaType),
								Reason: fmt.Sprintf("set remote description error: %v", err),
							},
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

		case *rtc.Signalling_Trickle:

			var candidate webrtc.ICECandidateInit
			err := json.Unmarshal([]byte(payload.Trickle.Init), &candidate)
			if err != nil {
				log.Errorf("error parsing ice candidate, error -> %v", err)
				err = stream.Send(&rtc.Signalling{
					Payload: &rtc.Signalling_Error{
						Error: &rtc.Error{
							Code:   int32(error_code.InternalError),
							Reason: fmt.Sprintf("unmarshal ice candidate error:  %v", err),
						},
					},
				})
				if err != nil {
					log.Errorf("grpc send error: %v", err)
					return status.Errorf(codes.Internal, err.Error())
				}
				continue
			}
			log.Infof("trickle: target %v, candidate %v", int(payload.Trickle.Target), candidate.Candidate)
			err = peer.Trickle(candidate, int(payload.Trickle.Target))
			if err != nil {
				switch err {
				case ion_sfu.ErrNoTransportEstablished:
					log.Errorf("peer hasn't joined, error -> %v", err)
					err = stream.Send(&rtc.Signalling{
						Payload: &rtc.Signalling_Error{
							Error: &rtc.Error{
								Code:   int32(error_code.InternalError),
								Reason: fmt.Sprintf("trickle error:  %v", err),
							},
						},
					})
					if err != nil {
						log.Errorf("grpc send error: %v", err)
						return status.Errorf(codes.Internal, err.Error())
					}
				default:
					return status.Errorf(codes.Unknown, fmt.Sprintf("negotiate error: %v", err))
				}
			}
		}
	}
}
