package rtc

import (
	"sync"
	"time"

	"github.com/pion/ion/pkg/log"
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
	pub         transport.Transport
	subs        map[string]transport.Transport
	subLock     sync.RWMutex
	stop        bool
	liveTime    time.Time
	pluginChain *plugins.PluginChain
}

// NewRouter return a new Router
func NewRouter(id string) *Router {
	log.Infof("NewRouter id=%s", id)
	return &Router{
		subs:        make(map[string]transport.Transport),
		liveTime:    time.Now().Add(liveCycle),
		pluginChain: plugins.NewPluginChain(),
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
			// log.Infof("pkt := <-r.subCh %v", pkt)
			if pkt == nil {
				continue
			}
			r.liveTime = time.Now().Add(liveCycle)
			// nonblock sending
			go func() {
				for _, t := range r.GetSubs() {
					if t == nil {
						log.Errorf("Transport is nil")
						continue
					}

					// log.Infof(" WriteRTP %v:%v to %v ", pkt.SSRC, pkt.SequenceNumber, t.ID())
					if err := t.WriteRTP(pkt); err != nil {
						// log.Errorf("wt.WriteRTP err=%v", err)
						// del sub when err is increasing
						if t.WriteErrTotal() > maxWriteErr {
							r.DelSub(t.ID())
						}
					}
					t.WriteErrReset()
				}
			}()
		}
	}()
}

// AddPub add a pub transport
func (r *Router) AddPub(id string, t transport.Transport) transport.Transport {
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

func MapRouter(fn func(id string, r *Router)) {
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

// AddSub add a pub to router
func (r *Router) AddSub(id string, t transport.Transport) transport.Transport {
	//fix panic: assignment to entry in nil map
	if r.stop {
		return nil
	}
	r.subLock.Lock()
	defer r.subLock.Unlock()
	r.subs[id] = t
	log.Infof("Router.AddSub id=%s t=%p", id, t)
	go func() {
		for {
			pkt := <-t.GetRTCPChan()
			if r.stop {
				return
			}
			switch pkt.(type) {
			case *rtcp.PictureLossIndication:
				if r.GetPub() != nil {
					// Request a Key Frame
					log.Infof("Router.AddSub got pli: %+v", pkt)
					r.GetPub().WriteRTCP(pkt)
				}
			case *rtcp.TransportLayerNack:
				// log.Infof("Router.AddSub got nack: %+v", pkt)
				nack := pkt.(*rtcp.TransportLayerNack)
				for _, nackPair := range nack.Nacks {
					if !r.ReSendRTP(id, nack.MediaSSRC, nackPair.PacketID) {
						n := &rtcp.TransportLayerNack{
							//origin ssrc
							SenderSSRC: nack.SenderSSRC,
							MediaSSRC:  nack.MediaSSRC,
							Nacks:      []rtcp.NackPair{rtcp.NackPair{PacketID: nackPair.PacketID}},
						}
						if r.pub != nil {
							r.GetPub().WriteRTCP(n)
						}
					}
				}

			default:
			}
		}
	}()
	return t
}

// GetSub get a sub by id
func (r *Router) GetSub(id string) transport.Transport {
	r.subLock.Lock()
	defer r.subLock.Unlock()
	// log.Infof("Router.GetSub id=%s sub=%v", id, r.subs[id])
	return r.subs[id]
}

// GetSubs get all subs
func (r *Router) GetSubs() map[string]transport.Transport {
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
func (r *Router) DelSub(id string) {
	log.Infof("Router.DelSub id=%s", id)
	r.subLock.Lock()
	defer r.subLock.Unlock()
	if r.subs[id] != nil {
		r.subs[id].Close()
	}
	delete(r.subs, id)
}

// DelSubs del all sub
func (r *Router) DelSubs() {
	log.Infof("Router.DelSubs")
	r.subLock.Lock()
	defer r.subLock.Unlock()
	for _, sub := range r.subs {
		if sub != nil {
			sub.Close()
		}
	}
	r.subs = nil
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

func (r *Router) ReSendRTP(sid string, ssrc uint32, sn uint16) bool {
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
	return !r.liveTime.Before(time.Now())
}
