package util

import (
	"bytes"
	"encoding/json"
	"net"
	"strings"

	"github.com/pion/ion/log"
	"github.com/pion/rtp"
	"github.com/pion/stun"
)

var (
	LocalIPPrefix = [...]string{"192.168", "10.0", "169.254", "172.16"}
)

func IsLocalIP(ip string) bool {
	for i := 0; i < len(LocalIPPrefix); i++ {
		if strings.HasPrefix(ip, LocalIPPrefix[i]) {
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

func GetInternetIP() string {

	// Creating a "connection" to STUN server.
	stunUrl := "stun.stunprotocol.org:3478"
	c, err := stun.Dial("udp", stunUrl)
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

func JsonEncode(str string) map[string]interface{} {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(str), &data); err != nil {
		panic(err)
	}
	return data
}

func Recover(flag string) {
	if err := recover(); err != nil {
		log.Errorf("[%s] recover err => %v", flag, err)
	}
}

// getValue get value from map
func GetValue(msg map[string]interface{}, key string) string {
	if msg == nil {
		return ""
	}
	id := msg[key]
	if id == nil {
		return ""
	}
	return id.(string)
}

// GetMap make kv to map, args should be multiple of 2
func GetMap(args ...interface{}) map[string]interface{} {
	if len(args)%2 != 0 {
		return nil
	}
	msg := make(map[string]interface{})
	for i := 0; i < len(args)/2; i++ {
		msg[args[2*i].(string)] = args[2*i+1]
	}
	return msg
}

func GetIDFromRTP(pkt *rtp.Packet) string {
	if !pkt.Header.Extension || len(pkt.Header.ExtensionPayload) < 36 {
		log.Warnf("pkt invalid extension")
		return ""
	}
	return string(bytes.TrimRight(pkt.Header.ExtensionPayload, "\x00"))
}

func SetIDToRTP(pkt *rtp.Packet, id string) *rtp.Packet {
	pkt.Header.Extension = true

	//the payload must be in 32-bit words and bigger than extPayload
	if len(pkt.Header.ExtensionPayload)%4 != 0 || len(pkt.Header.ExtensionPayload) < len(id) {
		n := 4 * (len(id)/4 + 1)
		pkt.Header.ExtensionPayload = make([]byte, n)
	}
	copy(pkt.Header.ExtensionPayload, id)
	return pkt
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
