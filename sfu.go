package main

import (
	"net/http"
	"time"

	"github.com/pion/webrtc/v2"

	_ "net/http/pprof"

	"github.com/pion/sfu/conf"
	"github.com/pion/sfu/gslb"
	"github.com/pion/sfu/log"
	"github.com/pion/sfu/service"
)

const (
	signalNameSpace = "signal:"
)

type ClientSDP struct {
	sdp      webrtc.SessionDescription
	clientID string
}

type SFU struct {
	gslb *gslb.GSLB
}

func NewSFU() (*SFU, error) {
	log.Infof("NewSFU")
	s := &SFU{}
	if !conf.Cfg.Mode.Single {
		g, err := gslb.New()
		if err != nil {
			return nil, err
		}
		s.gslb = g
	}
	service.StartSignalServer()
	return s, nil
}

func (s *SFU) Close() {
	s.gslb.Close()
}

func (s *SFU) Run() {
	for {
		time.Sleep(time.Second)
	}
}

func main() {
	// 开启pprof
	if conf.Cfg.Mode.Pprof != "" {
		go func() {
			http.ListenAndServe(conf.Cfg.Mode.Pprof, nil)
		}()
	}

	log.Infof("main")
	sfu, err := NewSFU()
	if err != nil {
		log.Panicf("err : %v", err)
	}
	sfu.Run()
}
