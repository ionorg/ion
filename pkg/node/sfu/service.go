package sfu

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/pion/ion-log"
	"github.com/bep/debounce"
	ion_sfu_log "github.com/pion/ion-sfu/pkg/logger"
	"github.com/pion/ion-sfu/pkg/middlewares/datachannel"
	"github.com/pion/ion-sfu/pkg/sfu"
	ion_sfu "github.com/pion/ion-sfu/pkg/sfu"
	error_code "github.com/pion/ion/pkg/error"
	"github.com/pion/ion/proto/rtc"
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var logrLogger = ion_sfu_log.New().WithName("ion-sfu-node")

func init() {
	ion_sfu_log.SetGlobalOptions(ion_sfu_log.GlobalConfig{V: 1})
	sfu.Logger = logrLogger.WithName("sfu")
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

func (s *SFUService) BroadcastTrackEvent(uid string, tracks []*rtc.TrackInfo, state rtc.TrackEvent_State) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	for id, sig := range s.sigs {
		if id == uid {
			continue
		}
		err := sig.Send(&rtc.Reply{
			Payload: &rtc.Reply_TrackEvent{
				TrackEvent: &rtc.TrackEvent{
					Uid:    uid,
					Tracks: tracks,
					State:  state,
				},
			},
		})
		if err != nil {
			log.Errorf("signal send error: %v", err)
		}
	}
}

func (s *SFUService) Signal(sig rtc.RTC_SignalServer) error {
	//val := sigStream.Context().Value("claims")
	//log.Infof("context val %v", val)
	peer := ion_sfu.NewPeer(s.sfu)
	var tracksMutex sync.RWMutex
	var tracksInfo []*rtc.TrackInfo

	defer func() {
		if peer.Session() != nil {
			log.Infof("[S=>C] close: sid => %v, uid => %v", peer.Session().ID(), peer.ID())
			uid := peer.ID()

			s.mutex.Lock()
			delete(s.sigs, peer.ID())
			s.mutex.Unlock()

			tracksMutex.Lock()
			defer tracksMutex.Unlock()
			if len(tracksInfo) > 0 {
				s.BroadcastTrackEvent(uid, tracksInfo, rtc.TrackEvent_REMOVE)
				log.Infof("broadcast tracks event %v, state = REMOVE", tracksInfo)
			}

			// Remove down tracks that other peers subscribed from this peer
			for _, downTrack := range peer.Subscriber().DownTracks() {
				streamID := downTrack.StreamID()
				for _, t := range tracksInfo {
					if downTrack != nil && downTrack.ID() == t.Id {
						log.Infof("remove down track[%v] from peer[%v]", downTrack.ID(), peer.ID())
						peer.Subscriber().RemoveDownTrack(streamID, downTrack)
						_ = downTrack.Stop()
					}
				}
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

			err = sig.Send(&rtc.Reply{
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
			if err != nil {
				log.Errorf("signal send error: %v", err)
			}

			publisher := peer.Publisher()

			if publisher != nil {
				var once sync.Once
				publisher.OnPublisherTrack(func(pt ion_sfu.PublisherTrack) {
					log.Debugf("[S=>C] OnPublisherTrack: \nKind %v, \nUid: %v,  \nMsid: %v,\nTrackID: %v", pt.Track.Kind(), uid, pt.Track.Msid(), pt.Track.ID())

					once.Do(func() {
						debounced := debounce.New(800 * time.Millisecond)
						debounced(func() {
							var peerTracks []*rtc.TrackInfo
							pubTracks := publisher.PublisherTracks()
							if len(pubTracks) == 0 {
								return
							}

							for _, pubTrack := range publisher.PublisherTracks() {
								peerTracks = append(peerTracks, &rtc.TrackInfo{
									Id:       pubTrack.Track.ID(),
									Kind:     pubTrack.Track.Kind().String(),
									StreamId: pubTrack.Track.StreamID(),
									Muted:    false,
									Layer:    pubTrack.Track.RID(),
								})
							}

							// broadcast the existing tracks in the session
							tracksInfo = append(tracksInfo, peerTracks...)
							log.Infof("[S=>C] BroadcastTrackEvent existing track %v, state = ADD", peerTracks)
							s.BroadcastTrackEvent(uid, peerTracks, rtc.TrackEvent_ADD)
							if err != nil {
								log.Errorf("signal send error: %v", err)
							}
						})
					})
				})
			}

			for _, p := range peer.Session().Peers() {
				var peerTracks []*rtc.TrackInfo
				if peer.ID() != p.ID() {
					pubTracks := p.Publisher().PublisherTracks()
					if len(pubTracks) == 0 {
						continue
					}

					for _, pubTrack := range p.Publisher().PublisherTracks() {
						peerTracks = append(peerTracks, &rtc.TrackInfo{
							Id:       pubTrack.Track.ID(),
							Kind:     pubTrack.Track.Kind().String(),
							StreamId: pubTrack.Track.StreamID(),
							Muted:    false,
							Layer:    pubTrack.Track.RID(),
						})
					}

					event := &rtc.TrackEvent{
						Uid:    p.ID(),
						State:  rtc.TrackEvent_ADD,
						Tracks: peerTracks,
					}

					// Send the existing tracks in the session to the new joined peer
					log.Infof("[S=>C] send existing track %v, state = ADD", peerTracks)
					err = sig.Send(&rtc.Reply{
						Payload: &rtc.Reply_TrackEvent{
							TrackEvent: event,
						},
					})
					if err != nil {
						log.Errorf("signal send error: %v", err)
					}
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
			log.Debugf("[C=>S] subscription: %v", payload.Subscription)
			subscription := payload.Subscription
			needNegotiate := false
			for _, trackInfo := range subscription.Subscriptions {
				if trackInfo.Subscribe {
					// Add down tracks
					for _, p := range peer.Session().Peers() {
						if p.ID() != peer.ID() {
							for _, track := range p.Publisher().PublisherTracks() {
								if track.Receiver.TrackID() == trackInfo.TrackId && track.Track.RID() == trackInfo.Layer {
									log.Infof("Add RemoteTrack: %v to peer %v %v %v", trackInfo.TrackId, peer.ID(), track.Track.Kind(), track.Track.RID())
									dt, err := peer.Publisher().GetRouter().AddDownTrack(peer.Subscriber(), track.Receiver)
									if err != nil {
										log.Errorf("AddDownTrack error: %v", err)
									}
									// switchlayer
									switch trackInfo.Layer {
									case "f":
										dt.Mute(false)
										_ = dt.SwitchSpatialLayer(2, true)
										log.Infof("%v SwitchSpatialLayer:  2", trackInfo.TrackId)
									case "h":
										dt.Mute(false)
										_ = dt.SwitchSpatialLayer(1, true)
										log.Infof("%v SwitchSpatialLayer:  1", trackInfo.TrackId)
									case "q":
										dt.Mute(false)
										_ = dt.SwitchSpatialLayer(0, true)
										log.Infof("%v SwitchSpatialLayer:  0", trackInfo.TrackId)
									}
									needNegotiate = true
								}
							}
						}
					}
				} else {
					// Remove down tracks
					for _, downTrack := range peer.Subscriber().DownTracks() {
						streamID := downTrack.StreamID()
						if downTrack != nil && downTrack.ID() == trackInfo.TrackId {
							peer.Subscriber().RemoveDownTrack(streamID, downTrack)
							_ = downTrack.Stop()
							needNegotiate = true
						}
					}
				}
			}
			if needNegotiate {
				peer.Subscriber().Negotiate()
			}

			_ = sig.Send(&rtc.Reply{
				Payload: &rtc.Reply_Subscription{
					Subscription: &rtc.SubscriptionReply{
						Success: true,
						Error:   nil,
					},
				},
			})
		}
	}
}
