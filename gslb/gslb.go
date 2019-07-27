package gslb

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pion/ion/conf"
	"github.com/pion/ion/log"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

var (
	ip       = conf.Global.AdveritiseIP
	port     = conf.Rtp.Port
	etcdBase = "ion://"
	etcdRoom = etcdBase + "room"
	etcdRtp  = etcdBase + "rtp"
	etcdNode = etcdBase + "node/" + ip + ":" + strconv.Itoa(port)
	etcd     *Etcd
	quit     chan struct{}
)

func init() {
	var err error
	etcd, err = newEtcd(conf.Etcd.Servers)
	if err != nil {
		panic(err)
	}
	quit = make(chan struct{})
	updateLoad()
}

func updateLoad() {
	go func() {
		etcd.keep(etcdNode, "")
		for {
			select {
			case <-quit:
				return
			case <-time.After(time.Second):
				etcd.update(etcdNode, getScore())
			}
		}
	}()
}

// cost 1 second
func getScore() string {
	var score float64
	p, err := cpu.Percent(time.Second, false)
	if len(p) != 1 {
		log.Errorf("cpu.Percent err => %v", err)
		return "0"
	}

	cpuScore := 100 - p[0]

	v, _ := mem.VirtualMemory()
	memScore := 100 - v.UsedPercent

	// test net by etcd
	var netScore float64
	baseTime := time.Now()
	_, err = etcd.get("netscore")
	costTime := time.Since(baseTime).Nanoseconds() / 1e6

	if err != nil {
		netScore = 0
	} else if costTime < 300 {
		netScore = float64(300-costTime) / 300 * 100
	} else if costTime >= 300 && costTime <= 1000 {
		netScore = float64(1000-costTime) / float64(1000) * 50
	} else {
		netScore = 0
	}

	score = cpuScore*0.4 + memScore*0.2 + netScore*0.4
	if cpuScore < 10 || memScore < 10 || netScore < 10 {
		score = 0
	}
	return strconv.Itoa(int(score))
}

func Close() {
	close(quit)
}

func KeepMediaInfo(rid, id string, payload uint8, ssrc uint32) error {
	log.Infof("gslb.KeepMediaInfo %s/%s/%s:%d/%s/%d %d", etcdRtp, rid, ip, conf.Rtp.Port, id, payload, ssrc)
	return etcd.keep(fmt.Sprintf("%s/%s/%s:%d/%s/%d", etcdRtp, rid, ip, conf.Rtp.Port, id, payload), fmt.Sprintf("%d", ssrc))
}

//ion://rtp/room1/127.0.0.1:6666/4ee0122d-464b-439c-b1dc-2ac72fb539a1/111
// 1925858818
func GetMediaInfo(rid, pid string) map[uint8]uint32 {
	kvs, _ := etcd.getByPrefix(etcdRtp + "/" + rid + "/")
	m := make(map[uint8]uint32)
	for k, ssrc := range kvs {
		if !strings.Contains(k, pid) {
			continue
		}
		if len(k) > len(etcdRtp+"/") {
			strs := strings.Split(k, "/")
			if len(strs) == 7 {
				payload, _ := strconv.Atoi(strs[6])
				nssrc, _ := strconv.Atoi(ssrc)
				m[uint8(payload)] = uint32(nssrc)
			}
		}
	}
	return m
}

//ion://rtp/room1/127.0.0.1:6666/4ee0122d-464b-439c-b1dc-2ac72fb539a1/111
// 1925858818
func GetPubs(rid string) map[string]string {
	kvs, _ := etcd.getByPrefix(etcdRtp + "/" + rid + "/")

	m := make(map[string]string)
	for k := range kvs {
		if len(k) > len(etcdRtp+"/") {
			strs := strings.Split(k, "/")
			if len(strs) == 7 {
				m[strs[5]] = strs[5]
			}
		}
	}
	log.Infof("GetPubs %v", m)
	return m
}

func getPubKey(rid, pid string) string {
	return etcdRoom + "/" + rid + "/pub/" + pid
}

func getSubKey(rid, sid string) string {
	return etcdRoom + "/" + rid + "/sub/" + sid
}

func PubWatch(rid, pid string, call WatchCallback) {
	go etcd.watch(getPubKey(rid, pid), call, true)
}

func NotifyPub(rid, pid string, msg map[string]interface{}) {
	log.Infof("NotifyPub rid=%s pid=%s  msg=%v", rid, pid, msg)
	byte, _ := json.Marshal(msg)
	key := getPubKey(rid, pid)
	log.Infof("key=%s byte=%s", key, byte)
	etcd.keep(key, string(byte))
}

func SubWatch(rid, sid string, call WatchCallback) {
	key := getSubKey(rid, sid)
	log.Infof("SubWatch key=%s", key)
	etcd.keep(key, "")
	go etcd.watch(key, call, false)
}

func DelWatch(rid, sid string) {
	key := getSubKey(rid, sid)
	log.Infof("DelWatch key=%s", key)
	etcd.del(key)
}

func NotifySubs(rid string, msg map[string]interface{}) {
	log.Infof("NotifySub rid=%s msg=%v", rid, msg)
	key := etcdRoom + "/" + rid + "/sub/"
	m, err := etcd.getByPrefix(key)
	log.Infof("etcd.GetByPrefix key=%s m=%v", key, m)
	if err != nil {
		log.Errorf("etcd.GetByPrefix err=%v", err)
		return
	}
	for k, _ := range m {
		msg["id"] = GetSubID([]byte(k))
		data, _ := json.Marshal(msg)
		log.Infof("etcd.Keep k=%s msg=%v data=%s", k, msg, data)
		etcd.keep(k, string(data))
	}
}

func GetPubID(data []byte) string {
	strs := strings.Split(string(data), "/")
	//ion://room/room1/pub/5a8774fb-aef8-411d-a1e5-8d30774f2d1f
	if len(strs) == 6 {
		return strs[5]
	}
	return ""
}

func GetSubID(data []byte) string {
	strs := strings.Split(string(data), "/")
	//ion://room/room1/sub/c7b1a783-5564-4ebe-9826-bf094a07bceb
	if len(strs) == 6 {
		return strs[5]
	}
	return ""
}

func IsPub(rid, pid string) bool {
	key := getPubKey(rid, pid)
	if _, err := etcd.get(key); err != nil {
		return false
	}
	return true
}
