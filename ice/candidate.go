package ice

import (
	"net"
	"time"
)

const (
	receiveMTU             = 8192
	defaultLocalPreference = 65535

	// ComponentRTP indicates that the candidate is used for RTP
	ComponentRTP uint16 = 1
	// ComponentRTCP indicates that the candidate is used for RTCP
	ComponentRTCP
)

// Candidate represents an ICE candidate
type Candidate interface {
	ID() string
	Component() uint16
	Address() string
	LastReceived() time.Time
	LastSent() time.Time
	NetworkType() NetworkType
	Port() int
	Priority() uint32
	RelatedAddress() *CandidateRelatedAddress
	String() string
	Type() CandidateType

	Equal(other Candidate) bool

	addr() *net.UDPAddr
	agent() *Agent

	close() error
	seen(outbound bool)
	start(a *Agent, conn net.PacketConn)
	writeTo(raw []byte, dst Candidate) (int, error)
}
