package sfu

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	log "github.com/pion/ion-log"
	ion_sfu_log "github.com/pion/ion-sfu/pkg/logger"
	"github.com/pion/ion-sfu/pkg/middlewares/datachannel"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	error_code "github.com/pion/ion/pkg/error"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func init() {
	ion_sfu_log.SetGlobalOptions(ion_sfu_log.GlobalConfig{V: 1})
}

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

func (s *SFUService) BroadcastStreamEvent(uid string, tracks []*rtc.Track, state rtc.TrackEvent_State) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for _, sig := range s.sigs {
		sig.Send(&rtc.Reply{
			Payload: &rtc.Reply_TrackEvent{
				TrackEvent: &rtc.TrackEvent{
					Uid:    uid,
					Tracks: tracks,
					State:  state,
				},
			},
		})
	}
}

func (s *SFUService) Signal(sig rtc.RTC_SignalServer) error {
	//val := sigStream.Context().Value("claims")
	//log.Infof("context val %v", val)

	peer := ion_sfu.NewPeer(s.sfu)
	var tracks []*rtc.Track
	var pubTracks []ion_sfu.PublisherTrack

	defer func() {
		if peer.Session() != nil {
			log.Infof("[S=>C] close: sid => %v, uid => %v", peer.Session().ID(), peer.ID())
			uid := peer.ID()

			s.mutex.Lock()
			delete(s.sigs, peer.ID())
			s.mutex.Unlock()

			if len(tracks) > 0 {
				s.BroadcastStreamEvent(uid, tracks, rtc.TrackEvent_REMOVE)
				log.Infof("broadcast tracks event %v, state = REMOVE", tracks)
			}
		}
	}()

	for {
		in, err := sig.Recv()

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
		case *rtc.Request_Join:
			sid := payload.Join.Sid
			uid := payload.Join.Uid
			log.Infof("[C=>S] join: sid => %v, uid => %v", sid, uid)

			//TODO: check auth info.

			// Notify user of new ice candidate
			peer.OnIceCandidate = func(candidate *webrtc.ICECandidateInit, target int) {
				log.Debugf("[S=>C] peer.OnIceCandidate: target = %v, candidate = %v", target, candidate.Candidate)
				bytes, err := json.Marshal(candidate)
				if err != nil {
					log.Errorf("OnIceCandidate error: %v", err)
				}
				err = sig.Send(&rtc.Reply{
					Payload: &rtc.Reply_Trickle{
						Trickle: &rtc.Trickle{
							Init:   string(bytes),
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
				log.Debugf("[S=>C] peer.OnOffer: %v", o.SDP)
				err = sig.Send(&rtc.Reply{
					Payload: &rtc.Reply_Description{
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
			nopub := false
			if val, found := payload.Join.Config["NoPublish"]; found {
				nopub = val == "true"
			}

			nosub := false
			if val, found := payload.Join.Config["NoSubscribe"]; found {
				nosub = val == "true"
			}

			noautosub := false
			if val, found := payload.Join.Config["NoAutoSubscribe"]; found {
				noautosub = val == "true"
			}

			cfg := ion_sfu.JoinConfig{
				NoPublish:       nopub,
				NoSubscribe:     nosub,
				NoAutoSubscribe: noautosub,
			}

			err = peer.Join(sid, uid, cfg)
			if err != nil {
				switch err {
				case ion_sfu.ErrTransportExists:
					fallthrough
				case ion_sfu.ErrOfferIgnored:
					err = sig.Send(&rtc.Reply{
						Payload: &rtc.Reply_Join{
							Join: &rtc.JoinReply{
								Success: false,
								Error: &rtc.Error{
									Code:   int32(error_code.InternalError),
									Reason: fmt.Sprintf("join error: %v", err),
								},
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

			desc := webrtc.SessionDescription{
				SDP:  payload.Join.Description.Sdp,
				Type: webrtc.NewSDPType(payload.Join.Description.Type),
			}

			log.Debugf("[C=>S] join.description: offer %v", desc.SDP)
			answer, err := peer.Answer(desc)
			if err != nil {
				return status.Errorf(codes.Internal, fmt.Sprintf("answer error: %v", err))
			}

			// send answer
			log.Debugf("[S=>C] join.description: answer %v", answer.SDP)

			sig.Send(&rtc.Reply{
				Payload: &rtc.Reply_Join{
					Join: &rtc.JoinReply{
						Success: true,
						Error:   nil,
						Description: &rtc.SessionDescription{
							Target: rtc.Target(rtc.Target_PUBLISHER),
							Sdp:    answer.SDP,
							Type:   answer.Type.String(),
						},
					},
				},
			})

			publisher := peer.Publisher()

			if publisher != nil {
				publisher.OnPublisherTrack(func(pt ion_sfu.PublisherTrack) {
					log.Debugf("[S=>C] OnPublisherTrack: \nKind %v, \nUid: %v,  \nMsid: %v,\nTrackID: %v", pt.Track.Kind(), uid, pt.Track.Msid(), pt.Track.ID())
					track := &rtc.Track{
						Id:       pt.Track.ID(),
						StreamId: pt.Track.StreamID(),
						Kind:     pt.Track.Kind().String(),
						Muted:    false,
						Rid:      pt.Track.RID(),
					}
					log.Infof("[S=>C] broadcast track %v, state = ADD", track)
					s.BroadcastStreamEvent(uid, []*rtc.Track{track}, rtc.TrackEvent_ADD)
					tracks = append(tracks, track)
					pubTracks = append(pubTracks, pt)
				})
			}

			for _, p := range peer.Session().Peers() {
				var peerTracks []*rtc.Track
				if peer.ID() != p.ID() {
					for _, pubTrack := range p.Publisher().PublisherTracks() {
						peerTracks = append(peerTracks, &rtc.Track{
							Id:       pubTrack.Track.ID(),
							Kind:     pubTrack.Track.Kind().String(),
							StreamId: pubTrack.Track.StreamID(),
							Muted:    false,
							Rid:      pubTrack.Track.RID(),
						})
					}

					event := &rtc.TrackEvent{
						Uid:    p.ID(),
						State:  rtc.TrackEvent_ADD,
						Tracks: peerTracks,
					}

					// Send the existing tracks in the session to the new joined peer
					log.Infof("[S=>C] send existing track %v, state = ADD", peerTracks)
					sig.Send(&rtc.Reply{
						Payload: &rtc.Reply_TrackEvent{
							TrackEvent: event,
						},
					})
				}
			}

			//TODO: Return error when the room is full, or locked, or permission denied

			s.mutex.Lock()
			s.sigs[peer.ID()] = sig
			s.mutex.Unlock()

		case *rtc.Request_Description:
			desc := webrtc.SessionDescription{
				SDP:  payload.Description.Sdp,
				Type: webrtc.NewSDPType(payload.Description.Type),
			}
			var err error = nil
			switch desc.Type {
			case webrtc.SDPTypeOffer:
				log.Debugf("[C=>S] description: offer %v", desc.SDP)
				answer, err := peer.Answer(desc)
				if err != nil {
					return status.Errorf(codes.Internal, fmt.Sprintf("answer error: %v", err))
				}

				// send answer
				log.Debugf("[S=>C] description: answer %v", answer.SDP)

				err = sig.Send(&rtc.Reply{
					Payload: &rtc.Reply_Description{
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

			case webrtc.SDPTypeAnswer:
				log.Debugf("[C=>S] description: answer %v", desc.SDP)
				err = peer.SetRemoteDescription(desc)
			}

			if err != nil {
				switch err {
				case ion_sfu.ErrNoTransportEstablished:
					err = sig.Send(&rtc.Reply{
						Payload: &rtc.Reply_Join{
							Join: &rtc.JoinReply{
								Success: false,
								Error: &rtc.Error{
									Code:   int32(error_code.UnsupportedMediaType),
									Reason: fmt.Sprintf("set remote description error: %v", err),
								},
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

		case *rtc.Request_Trickle:
			var candidate webrtc.ICECandidateInit
			err := json.Unmarshal([]byte(payload.Trickle.Init), &candidate)
			if err != nil {
				log.Errorf("error parsing ice candidate, error -> %v", err)
				err = sig.Send(&rtc.Reply{
					Payload: &rtc.Reply_Error{
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
			log.Debugf("[C=>S] trickle: target %v, candidate %v", int(payload.Trickle.Target), candidate.Candidate)
			err = peer.Trickle(candidate, int(payload.Trickle.Target))
			if err != nil {
				switch err {
				case ion_sfu.ErrNoTransportEstablished:
					log.Errorf("peer hasn't joined, error -> %v", err)
					err = sig.Send(&rtc.Reply{
						Payload: &rtc.Reply_Error{
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

		case *rtc.Request_Subscription:
			subscription := payload.Subscription
			subscribe := subscription.GetSubscribe()
			needNegotiate := false
			for _, trackId := range subscription.TrackIds {
				if subscribe {
					// Add down tracks
					for _, p := range peer.Session().Peers() {
						if p.ID() != peer.ID() {
							for _, track := range p.Publisher().PublisherTracks() {
								if track.Receiver.TrackID() == trackId {
									log.Debugf("Add RemoteTrack: %v to peer %v", trackId, peer.ID())
									peer.Publisher().GetRouter().AddDownTrack(peer.Subscriber(), track.Receiver)
									needNegotiate = true
								}
							}
						}
					}
				} else {
					// Remove down tracks
					for _, downTrack := range peer.Subscriber().DownTracks() {
						streamID := downTrack.StreamID()
						if downTrack != nil && downTrack.ID() == trackId {
							peer.Subscriber().RemoveDownTrack(streamID, downTrack)
							downTrack.Stop()
							needNegotiate = true
						}
					}
				}
			}
			if needNegotiate {
				peer.Subscriber().Negotiate()
			}
		}
	}
}
