package rtc

import (
	"errors"
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
	maxSize     = 1024
	jbPlugin    = "jitterBuffer"
	liveCycle   = 6 * time.Second
)

var (
	errInvalidPlugin = errors.New("plugin is nil")
)

// Router is a rtp Router
//                                    +--->sub
//                                    |
// pub--->pubCh-->plugin...-->subCh---+--->sub
//                                    |
//                                    +--->sub

type rtcpInfo struct {
	id string
	rtcp.Packet
}

// Router is rtp router
type Router struct {
	pub        transport.Transport
	subs       map[string]transport.Transport
	subLock    sync.RWMutex
	plugins    []plugins.Plugin
	pluginLock sync.RWMutex
	pubCh      chan *rtp.Packet
	subCh      chan *rtp.Packet
	stop       bool
	liveTime   time.Time
	jbRtcpCh   chan rtcp.Packet
	jbConfig   plugins.JitterBufferConfig
}

// NewRouter return a new Router
func NewRouter(id string) *Router {
	log.Infof("NewRouter id=%s", id)
	jb := plugins.NewJitterBuffer(jbPlugin)
	r := &Router{
		subs:     make(map[string]transport.Transport),
		pubCh:    make(chan *rtp.Packet, maxSize),
		subCh:    make(chan *rtp.Packet, maxSize),
		liveTime: time.Now().Add(liveCycle),
		jbRtcpCh: jb.GetRTCPChan(),
	}
	r.AddPlugin(jbPlugin, jb)
	r.start()
	return r
}

func (r *Router) in() {
	go func() {
		defer util.Recover("[Router.in]")
		count := uint64(0)
		for {
			if r.stop {
				return
			}
			pub := r.GetPub()
			if pub == nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			rtp, err := pub.ReadRTP()
			if err == nil {
				// log.Infof("rtp.Extension=%t rtp.ExtensionProfile=%x rtp.ExtensionPayload=%x", rtp.Extension, rtp.ExtensionProfile, rtp.ExtensionPayload)
				r.pubCh <- rtp
				if count%300 == 0 {
					r.liveTime = time.Now().Add(liveCycle)
				}
				count++
			} else {
				log.Errorf("Router.in err=%v", err)
			}
		}
	}()
}

func (r *Router) handle() {
	go func() {
		defer util.Recover("[Router.handle]")
		count := uint64(0)
		for {
			if r.stop {
				return
			}

			pkt := <-r.pubCh
			log.Debugf("pkt := <-r.pubCh %v", pkt)
			r.subCh <- pkt
			log.Debugf("r.subCh <- pkt %v", pkt)
			if pkt == nil {
				continue
			}
			//only buffer video
			if util.IsVideo(pkt.PayloadType) {
				if count%3000 == 0 {
					r.GetPlugin(jbPlugin).(*plugins.JitterBuffer).Init(pkt.SSRC, pkt.PayloadType, r.jbConfig)
				}
				r.GetPlugin(jbPlugin).PushRTP(pkt)
				count++
			}
		}
	}()
}

func (r *Router) out() {
	go func() {
		defer util.Recover("[Router.out]")
		for {
			if r.stop {
				return
			}

			pkt := <-r.subCh
			log.Debugf("pkt := <-r.subCh %v", pkt)
			if pkt == nil {
				continue
			}
			// nonblock sending
			go func() {
				for _, t := range r.GetSubs() {
					if t == nil {
						log.Errorf("Transport is nil")
						continue
					}

					// log.Infof("Router.out WriteRTP %v:%v to %v ", pkt.SSRC, pkt.SequenceNumber, t.ID())
					if err := t.WriteRTP(pkt); err != nil {
						log.Errorf("wt.WriteRTP err=%v", err)
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

func (r *Router) jitter() {
	go func() {
		defer util.Recover("[Router.out]")
		for {
			if r.stop {
				return
			}

			pkt := <-r.jbRtcpCh
			switch pkt.(type) {
			case *rtcp.TransportLayerNack, *rtcp.ReceiverEstimatedMaximumBitrate, *rtcp.PictureLossIndication:
				log.Infof("Router.jitter r.GetPub().WriteRTCP %v", pkt)
				if r.pub != nil {
					r.GetPub().WriteRTCP(pkt)
				}
			}
		}
	}()
}

func (r *Router) start() {
	r.in()
	r.out()
	r.handle()
	r.jitter()
}

// AddPub add a pub transport
func (r *Router) AddPub(id string, t transport.Transport) transport.Transport {
	log.Infof("AddPub id=%s", id)
	r.pub = t
	r.jbConfig = plugins.JitterBufferConfig{
		RembCycle: 2,
		PliCycle:  1,
		Bandwidth: t.GetBandwidth(),
	}
	return t
}

// DelPub del pub
func (r *Router) DelPub() {
	log.Infof("Router.DelPub %v", r.pub)
	// first close pub
	if r.pub != nil {
		r.pub.Close()
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
				if r.pub != nil {
					// Request a Key Frame
					log.Infof("Router.AddSub got pli: %+v", pkt)
					r.GetPub().WriteRTCP(pkt)
				}
			case *rtcp.TransportLayerNack:
				log.Debugf("rtptransport got nack: %+v", pkt)
				nack := pkt.(*rtcp.TransportLayerNack)
				for _, nackPair := range nack.Nacks {
					if !r.writeRTP(id, nack.MediaSSRC, nackPair.PacketID) {
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
				r.PushRTCP(pkt)
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

// AddPlugin add a plugin
func (r *Router) AddPlugin(id string, i plugins.Plugin) {
	r.pluginLock.Lock()
	defer r.pluginLock.Unlock()
	r.plugins = append(r.plugins, i)
}

// GetPlugin get plugin by id
func (r *Router) GetPlugin(id string) plugins.Plugin {
	r.pluginLock.RLock()
	defer r.pluginLock.RUnlock()
	for i := 0; i < len(r.plugins); i++ {
		if r.plugins[i].ID() == id {
			return r.plugins[i]
		}
	}
	return nil
}

// DelPlugin del plugin
func (r *Router) DelPlugin(id string) {
	r.pluginLock.Lock()
	defer r.pluginLock.Unlock()
	for i := 0; i < len(r.plugins); i++ {
		if r.plugins[i].ID() == id {
			r.plugins[i].Stop()
			r.plugins = append(r.plugins[:i], r.plugins[i+1:]...)
		}
	}
}

// DelPlugins del all plugins
func (r *Router) DelPlugins() {
	r.pluginLock.Lock()
	defer r.pluginLock.Unlock()
	for _, plugin := range r.plugins {
		plugin.Stop()
	}
}

// Close release all
func (r *Router) Close() {
	if r.stop {
		return
	}
	r.DelPub()
	r.stop = true
	r.DelPlugins()
	r.DelSubs()
}

func (r *Router) writeRTP(sid string, ssrc uint32, sn uint16) bool {
	if r.pub == nil {
		return false
	}
	hd := r.GetPlugin(jbPlugin)
	if hd != nil {
		jb := hd.(*plugins.JitterBuffer)
		pkt := jb.GetPacket(ssrc, sn)
		if pkt == nil {
			// log.Infof("Router.writeRTP pkt not found sid=%s ssrc=%d sn=%d pkt=%v", sid, ssrc, sn, pkt)
			return false
		}
		sub := r.GetSub(sid)
		if sub != nil {
			sub.WriteRTP(pkt)
			// log.Infof("Router.writeRTP sid=%s ssrc=%d sn=%d", sid, ssrc, sn)
			return true
		}
	}
	return false
}

// Alive return router status
func (r *Router) Alive() bool {
	if r.liveTime.Before(time.Now()) {
		return false
	}
	return true
}

// PushRTCP push rtcp to jitterbuffer
func (r *Router) PushRTCP(pkt rtcp.Packet) error {
	jbPlugin := r.GetPlugin(jbPlugin)
	if jbPlugin == nil {
		return errInvalidPlugin
	}
	return jbPlugin.PushRTCP(pkt)
}
