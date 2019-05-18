package util

import (
	"net"
	"strings"
)

func GetIP(internet bool) string {
	addrs, _ := net.InterfaceAddrs()
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if internet {
				if !strings.HasPrefix(ipnet.IP.String(), "192.168") && !strings.HasPrefix(ipnet.IP.String(), "10.0") {
					return ipnet.IP.String()
				}
			} else {
				if strings.HasPrefix(ipnet.IP.String(), "192.168") || strings.HasPrefix(ipnet.IP.String(), "10.0") {
					return ipnet.IP.String()
				}
			}
		}
	}
	return ""
}
