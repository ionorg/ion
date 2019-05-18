package main

import (
	"fmt"
	"net/http"

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
	room *service.Room
}

func NewSFU() (*SFU, error) {
	log.Infof("NewSFU")
	s := &SFU{}
	if !conf.Cfg.Mode.Standalone {
		g, err := gslb.New()
		if err != nil {
			return nil, err
		}
		s.gslb = g
	}

	s.room = service.NewRoom(signalNameSpace + "room1")

	return s, nil

}

func (s *SFU) Close() {
	s.gslb.Close()
}

func (s *SFU) Run() {
	s.room.Run()
}

func main() {
	// 开启pprof
	go func() {
		fmt.Println("pprof listen 6060")
		http.ListenAndServe(":6060", nil)
	}()

	log.Infof("main")
	sfu, err := NewSFU()
	if err != nil {
		log.Panicf("err : %v", err)
	}
	sfu.Run()
}
