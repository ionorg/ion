package main

import (
	"errors"
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
	// room *service.Room
	room interface{}
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
	if conf.Cfg.Mode.Signal == "protoo" {
		log.Infof("new sfu protoo")
		s.room = service.NewPRoom(signalNameSpace + "room1")
	} else if conf.Cfg.Mode.Signal == "centrifugo" {
		s.room = service.NewRoom(signalNameSpace + "room1")
	} else {
		return nil, errors.New("invalid signal type")
	}

	return s, nil

}

func (s *SFU) Close() {
	s.gslb.Close()
}

func (s *SFU) Run() {
	if conf.Cfg.Mode.Signal == "protoo" {
		s.room.(*service.PRoom).Run()
	} else if conf.Cfg.Mode.Signal == "centrifugo" {
		s.room.(*service.Room).Run()
	}
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
