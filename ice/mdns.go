package ice

import (
	"net"

	"github.com/sssgun/ion/logging"
	"github.com/sssgun/ion/mdns"
	"golang.org/x/net/ipv4"
)

// MulticastDNSMode represents the different Multicast modes ICE can run in
type MulticastDNSMode byte

// MulticastDNSMode enum
const (
	// MulticastDNSModeDisabled means remote mDNS candidates will be discarded, and local host candidates will use IPs
	MulticastDNSModeDisabled MulticastDNSMode = iota + 1

	// MulticastDNSModeQueryOnly means remote mDNS candidates will be accepted, and local host candidates will use IPs
	MulticastDNSModeQueryOnly

	// MulticastDNSModeQueryAndGather means remote mDNS candidates will be accepted, and local host candidates will use mDNS
	MulticastDNSModeQueryAndGather
)

func generateMulticastDNSName() (string, error) {
	return generateRandString("", ".local")
}

func createMulticastDNS(mDNSMode MulticastDNSMode, mDNSName string, log logging.LeveledLogger) (*mdns.Conn, MulticastDNSMode, error) {
	if mDNSMode == MulticastDNSModeDisabled {
		return nil, mDNSMode, nil
	}

	addr, mdnsErr := net.ResolveUDPAddr("udp4", mdns.DefaultAddress)
	if mdnsErr != nil {
		return nil, mDNSMode, mdnsErr
	}

	l, mdnsErr := net.ListenUDP("udp4", addr)
	if mdnsErr != nil {
		// If ICE fails to start MulticastDNS server just warn the user and continue
		log.Errorf("Failed to enable mDNS, continuing in mDNS disabled mode: (%s)", mdnsErr)
		return nil, MulticastDNSModeDisabled, nil
	}

	switch mDNSMode {
	case MulticastDNSModeQueryOnly:
		conn, err := mdns.Server(ipv4.NewPacketConn(l), &mdns.Config{})
		return conn, mDNSMode, err
	case MulticastDNSModeQueryAndGather:
		conn, err := mdns.Server(ipv4.NewPacketConn(l), &mdns.Config{
			LocalNames: []string{mDNSName},
		})
		return conn, mDNSMode, err
	default:
		return nil, mDNSMode, nil
	}
}
