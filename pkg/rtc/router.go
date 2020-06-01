package rtc

import (
	"sync"
	"time"

	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	"github.com/pion/ion/pkg/rtc/plugins"
	"github.com/pion/ion/pkg/rtc/transport"
	"github.com/pion/ion/pkg/util"
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

const (
	maxWriteErr = 100
	liveCycle   = 6 * time.Second
)

//                                      +--->sub
//                                      |
// pub--->pubCh-->pluginChain-->subCh---+--->sub
//                                      |
//                                      +--->sub
// Router is rtp router
type Router struct {
	pub           transport.Transport
	subs          map[proto.MID]transport.Transport
	subLock       sync.RWMutex
	stop          bool
	liveTime      time.Time
	pluginChain   *plugins.PluginChain
	subChans      map[proto.MID]chan *rtp.Packet
	subShutdownCh chan string
}

// NewRouter return a new Router
func NewRouter(id proto.MID) *Router {
	log.Infof("NewRouter id=%s", id)
	return &Router{
		subs:          make(map[proto.MID]transport.Transport),
		liveTime:      time.Now().Add(liveCycle),
		pluginChain:   plugins.NewPluginChain(string(id)),
		subChans:      make(map[proto.MID]chan *rtp.Packet),
		subShutdownCh: make(chan string, 1),
	}
}

func (r *Router) InitPlugins(config plugins.Config) error {
	log.Infof("Router.InitPlugins config=%+v", config)
	if r.pluginChain != nil {
		return r.pluginChain.Init(config)
	}
	return nil
}

func (r *Router) start() {
	go func() {
		defer util.Recover("[Router.start]")
		for {
			if r.stop {
				return
			}

			// Check sub cleanup
			select {
			case subID := <-r.subShutdownCh:
				log.Infof("Got transport shutdown %v", subID)
				r.DelSub(proto.MID(subID))
			default:
			}

			var pkt *rtp.Packet
			var err error
			// get rtp from pluginChain or pub
			if r.pluginChain != nil && r.pluginChain.On() {
				pkt = r.pluginChain.ReadRTP()
			} else {
				pkt, err = r.pub.ReadRTP()
				if err != nil {
					log.Errorf("r.pub.ReadRTP err=%v", err)
					continue
				}
			}
			// log.Debugf("pkt := <-r.subCh %v", pkt)
			if pkt == nil {
				continue
			}
			r.liveTime = time.Now().Add(liveCycle)
			r.subLock.RLock()
			// Push to client send queues
			for i := range r.GetSubs() {
				// Nonblock sending
				select {
				case r.subChans[i] <- pkt:
				default:
					log.Errorf("Sub consumer is backed up. Dropping packet")
				}
			}
			r.subLock.RUnlock()
		}
	}()
}

// AddPub add a pub transport
func (r *Router) AddPub(id proto.UID, t transport.Transport) transport.Transport {
	log.Infof("AddPub id=%s", id)
	r.pub = t
	r.pluginChain.AttachPub(t)
	r.start()
	return t
}

// DelPub del pub
func (r *Router) DelPub() {
	log.Infof("Router.DelPub %v", r.pub)
	if r.pub != nil {
		r.pub.Close()
	}
	if r.pluginChain != nil {
		r.pluginChain.Close()
	}
	r.pub = nil
}

func MapRouter(fn func(id proto.MID, r *Router)) {
	routerLock.RLock()
	defer routerLock.RUnlock()
	for id, r := range routers {
		fn(id, r)
	}
}

// GetPub get pub
func (r *Router) GetPub() transport.Transport {
	// log.Infof("Router.GetPub %v", r.pub)
	return r.pub
}

func (r *Router) subWriteLoop(subID proto.MID, trans transport.Transport) {
	for pkt := range r.subChans[subID] {
		// log.Infof(" WriteRTP %v:%v to %v PT: %v", pkt.SSRC, pkt.SequenceNumber, trans.ID(), pkt.Header.PayloadType)

		if err := trans.WriteRTP(pkt); err != nil {
			// log.Errorf("wt.WriteRTP err=%v", err)
			// del sub when err is increasing
			if trans.WriteErrTotal() > maxWriteErr {
				r.DelSub(proto.MID(trans.ID()))
			}
		}
		trans.WriteErrReset()
	}
	log.Infof("Closing sub writer")
}

func (r *Router) subFeedbackLoop(subID proto.MID, trans transport.Transport) {
	for pkt := range trans.GetRTCPChan() {
		if r.stop {
			break
		}
		switch pkt := pkt.(type) {
		case *rtcp.PictureLossIndication, *rtcp.FullIntraRequest:
			if r.GetPub() != nil {
				// Request a Key Frame
				log.Infof("Router got pli: %d", pkt.DestinationSSRC())
				err := r.GetPub().WriteRTCP(pkt)
				if err != nil {
					log.Errorf("Router pli err => %+v", err)
				}
			}
		case *rtcp.TransportLayerNack:
			// log.Infof("Router got nack: %+v", pkt)
			nack := pkt
			for _, nackPair := range nack.Nacks {
				if !r.ReSendRTP(subID, nack.MediaSSRC, nackPair.PacketID) {
					n := &rtcp.TransportLayerNack{
						//origin ssrc
						SenderSSRC: nack.SenderSSRC,
						MediaSSRC:  nack.MediaSSRC,
						Nacks:      []rtcp.NackPair{{PacketID: nackPair.PacketID}},
					}
					if r.pub != nil {
						err := r.GetPub().WriteRTCP(n)
						if err != nil {
							log.Errorf("Router nack WriteRTCP err => %+v", err)
						}
					}
				}
			}

		default:
		}
	}
	log.Infof("Closing sub feedback")
}

// AddSub add a sub to router
func (r *Router) AddSub(id proto.MID, t transport.Transport) transport.Transport {
	//fix panic: assignment to entry in nil map
	if r.stop {
		return nil
	}
	r.subLock.Lock()
	defer r.subLock.Unlock()
	r.subs[id] = t
	r.subChans[id] = make(chan *rtp.Packet, 1000)
	t.SetShutdownChan(r.subShutdownCh)
	log.Infof("Router.AddSub id=%s t=%p", id, t)

	// Sub loops
	go r.subWriteLoop(id, t)
	go r.subFeedbackLoop(id, t)
	return t
}

// GetSub get a sub by id
func (r *Router) GetSub(id proto.MID) transport.Transport {
	r.subLock.RLock()
	defer r.subLock.RUnlock()
	// log.Infof("Router.GetSub id=%s sub=%v", id, r.subs[id])
	return r.subs[id]
}

// GetSubs get all subs
func (r *Router) GetSubs() map[proto.MID]transport.Transport {
	r.subLock.RLock()
	defer r.subLock.RUnlock()
	// log.Infof("Router.GetSubs len=%v", len(r.subs))
	return r.subs
}

// HasNoneSub check if sub == 0
func (r *Router) HasNoneSub() bool {
	r.subLock.RLock()
	defer r.subLock.RUnlock()
	isNoSub := len(r.subs) == 0
	log.Infof("Router.HasNoneSub=%v", isNoSub)
	return isNoSub
}

// DelSub del sub by id
func (r *Router) DelSub(id proto.MID) {
	log.Infof("Router.DelSub id=%s", id)
	r.subLock.Lock()
	defer r.subLock.Unlock()
	if r.subs[id] != nil {
		r.subs[id].Close()
	}
	if r.subChans[id] != nil {
		close(r.subChans[id])
	}
	delete(r.subs, id)
	delete(r.subChans, id)
}

// DelSubs del all sub
func (r *Router) DelSubs() {
	log.Infof("Router.DelSubs")
	r.subLock.RLock()
	keys := make([]proto.MID, 0, len(r.subs))
	for k := range r.subs {
		keys = append(keys, k)
	}
	r.subLock.RUnlock()

	for _, id := range keys {
		r.DelSub(id)
	}
}

// Close release all
func (r *Router) Close() {
	if r.stop {
		return
	}
	log.Infof("Router.Close")
	r.DelPub()
	r.stop = true
	r.DelSubs()
}

func (r *Router) ReSendRTP(sid proto.MID, ssrc uint32, sn uint16) bool {
	if r.pub == nil {
		return false
	}
	hd := r.pluginChain.GetPlugin(plugins.TypeJitterBuffer)
	if hd != nil {
		jb := hd.(*plugins.JitterBuffer)
		pkt := jb.GetPacket(ssrc, sn)
		if pkt == nil {
			// log.Infof("Router.ReSendRTP pkt not found sid=%s ssrc=%d sn=%d pkt=%v", sid, ssrc, sn, pkt)
			return false
		}
		sub := r.GetSub(sid)
		if sub != nil {
			err := sub.WriteRTP(pkt)
			if err != nil {
				log.Errorf("router.ReSendRTP err=%v", err)
			}
			// log.Infof("Router.ReSendRTP sid=%s ssrc=%d sn=%d", sid, ssrc, sn)
			return true
		}
	}
	return false
}

// Alive return router status
func (r *Router) Alive() bool {
	return r.liveTime.After(time.Now())
}
