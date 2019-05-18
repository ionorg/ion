package gslb

import (
	"errors"
	"strconv"
	"time"

	"github.com/pion/sfu/conf"
	"github.com/pion/sfu/util"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

type GSLB struct {
	client *Client
	quit   chan bool
}

func New() (*GSLB, error) {
	c, err := NewClient(conf.Cfg.Etcd.Servers, "sfu://"+util.GetIP(false), "0")
	if err != nil {
		return nil, err
	}
	return &GSLB{
		client: c,
		quit:   make(chan bool),
	}, nil
}

func (g *GSLB) KeepAlive() error {
	if g == nil {
		return errors.New("gslb is nil")
	}
	g.client.KeepAlive()
	go g.UpdateLoad()
	return nil
}

func (g *GSLB) UpdateLoad() error {
	if g == nil {
		return errors.New("gslb is nil")
	}
	for {
		select {
		case <-g.quit:
			return nil
		case <-time.After(time.Second):
			ip := util.GetIP(true)
			if ip != "" {
				g.client.Update("sfu://"+ip, strconv.Itoa(g.getScore()))
			}
		}
	}
	return nil
}

// cost 1 second
func (g *GSLB) getScore() int {
	if g == nil {
		return -1
	}
	var score float64
	p, _ := cpu.Percent(time.Second, false)
	if len(p) != 1 {
		return 0
	}

	cpuScore := 100 - p[0]

	v, _ := mem.VirtualMemory()
	memScore := 100 - v.UsedPercent

	// test net by etcd
	var netScore float64
	baseTime := time.Now()
	_, err := g.client.Get("netscore")
	costTime := time.Since(baseTime).Nanoseconds() / 1e6

	if err != nil {
		netScore = 0
	} else if costTime < 200 {
		netScore = float64(200-costTime) / 200 * 100
	} else if costTime >= 100 && costTime <= 1000 {
		netScore = float64(1000-costTime) / float64(900) * 50
	} else {
		netScore = 0
	}

	score = cpuScore*0.4 + memScore*0.2 + netScore*0.4
	return int(score)
}

func (g *GSLB) Close() {
	if g.quit != nil {
		g.quit <- true
		close(g.quit)
	}
}
