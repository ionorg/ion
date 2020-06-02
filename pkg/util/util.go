package util

import (
	"encoding/json"
	"math/rand"
	"net"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"

	"time"

	nprotoo "github.com/cloudwebrtc/nats-protoo"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/stun"
)

var (
	localIPPrefix = [...]string{"192.168", "10.0", "169.254", "172.16"}
)

// KvOK check flag and value
func KvOK(m map[string]interface{}, k, v string) bool {
	str := ""
	val, ok := m[k]
	if ok {
		str, ok = val.(string)
		if ok {
			if strings.EqualFold(str, v) {
				return true
			}
		}
	}
	return false
}

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

func GetIPFromSTUN(stunURL string) string {
	// Creating a "connection" to STUN server.
	c, err := stun.Dial("udp", stunURL)
	if err != nil {
		log.Errorf("stun dial err %v", err)
		return ""
	}

	var ip string
	// Building binding request with random transaction id.
	message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
	// Sending request to STUN server, waiting for response message.
	ch := make(chan string)
	if err := c.Do(message, func(res stun.Event) {
		if res.Error != nil {
			log.Errorf("stun res err %v", err)
			close(ch)
			return
		}
		// Decoding XOR-MAPPED-ADDRESS attribute from message.
		var xorAddr stun.XORMappedAddress
		if err := xorAddr.GetFrom(res.Message); err != nil {
			log.Errorf("stun messge err %v", err)
			return
		}
		ip = xorAddr.IP.String()
	}); err != nil {
		log.Errorf("stun do err %v", err)
		close(ch)
		return ""
	}

	return ip
}

func MarshalStr(args ...interface{}) string {
	m := Map(args)
	if byt, err := json.Marshal(m); err != nil {
		log.Errorf(err.Error())
		return ""
	} else {
		return string(byt)
	}
}

func MarshalStrMap(m map[string]string) string {
	if byt, err := json.Marshal(m); err != nil {
		log.Errorf(err.Error())
		return ""
	} else {
		return string(byt)
	}
}

func Marshal(m map[string]interface{}) string {
	if byt, err := json.Marshal(m); err != nil {
		log.Errorf(err.Error())
		return ""
	} else {
		return string(byt)
	}
}

func Unmarshal(str string) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		log.Errorf(err.Error())
		return data
	}
	return data
}

func Recover(flag string) {
	_, _, l, _ := runtime.Caller(1)
	if err := recover(); err != nil {
		log.Errorf("[%s] Recover panic line => %v", flag, l)
		log.Errorf("[%s] Recover err => %v", flag, err)
		debug.PrintStack()
	}
}

// get value from map
func Val(msg map[string]interface{}, key string) string {
	if msg == nil {
		return ""
	}
	val := msg[key]
	if val == nil {
		return ""
	}
	switch val := val.(type) {
	case string:
		return val
	case map[string]interface{}:
		return Marshal(val)
	default:
		log.Errorf("util.Val val=%v", val)
		return ""
	}
}

// make kv to map, args should be multiple of 2
func Map(args ...interface{}) map[string]interface{} {
	if len(args)%2 != 0 {
		return nil
	}
	msg := make(map[string]interface{})
	for i := 0; i < len(args)/2; i++ {
		msg[args[2*i].(string)] = args[2*i+1]
	}
	return msg
}

func GetIP(addr string) string {
	if strings.Contains(addr, ":") {
		return strings.Split(addr, ":")[0]
	}
	return ""
}

func GetPort(addr string) string {
	if strings.Contains(addr, ":") {
		return strings.Split(addr, ":")[1]
	}
	return ""
}

func GetLostSN(begin, bitmap uint16) []uint16 {
	if bitmap == 0 {
		return []uint16{begin}
	}
	var sns []uint16
	for i := uint16(0); i < 15; i++ {
		if (bitmap >> i & 0x01) == 1 {
			sns = append(sns, begin+i)
		}
	}
	return sns
}

func GetMills() int64 {
	return time.Now().UnixNano() / 1e6
}

func StrToUint8(str string) uint8 {
	i, err := strconv.ParseUint(str, 10, 8)
	log.Infof("StrToUint8 str=%v i=%v err=%v", str, i, err)
	return uint8(i)
}

func StrToUint32(str string) uint32 {
	i, err := strconv.ParseUint(str, 10, 32)
	log.Infof("StrToUint32 str=%v i=%v err=%v", str, i, err)
	return uint32(i)
}

func randInt(min int, max int) int {
	return min + rand.Intn(max-min)
}

func RandStr(l int) string {
	bytes := make([]byte, l)
	for i := 0; i < l; i++ {
		bytes[i] = byte(randInt(65, 90))
	}
	return string(bytes)
}

func NewNpError(code int, reason string) *nprotoo.Error {
	err := nprotoo.Error{
		Code:   code,
		Reason: reason,
	}
	return &err
}
