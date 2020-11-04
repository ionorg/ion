package util

import (
	"net"
	"runtime"
	"runtime/debug"
	"strings"
	"sync/atomic"

	log "github.com/pion/ion-log"
)

var (
	localIPPrefix = [...]string{"192.168", "10.0", "169.254", "172.16"}
)

func IsLocalIP(ip string) bool {
	for i := 0; i < len(localIPPrefix); i++ {
		if strings.HasPrefix(ip, localIPPrefix[i]) {
			return true
		}
	}
	return false
}

func GetIntefaceIP() string {
	addrs, _ := net.InterfaceAddrs()

	// get internet ip first
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			if !IsLocalIP(ipnet.IP.String()) {
				return ipnet.IP.String()
			}
		}
	}

	// get internat ip
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
			return ipnet.IP.String()
		}
	}

	return ""
}

func Recover(flag string) {
	_, _, l, _ := runtime.Caller(1)
	if err := recover(); err != nil {
		log.Errorf("[%s] Recover panic line => %v", flag, l)
		log.Errorf("[%s] Recover err => %v", flag, err)
		debug.PrintStack()
	}
}

// AtomicBool represents a atomic bool
type AtomicBool struct {
	val int32
}

// Set automic bool
func (b *AtomicBool) Set(value bool) { // nolint: unparam
	var i int32
	if value {
		i = 1
	}

	atomic.StoreInt32(&(b.val), i)
}

// Get automic bool
func (b *AtomicBool) Get() bool {
	return atomic.LoadInt32(&(b.val)) != 0
}
