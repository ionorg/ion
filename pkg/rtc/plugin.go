package rtc

import (
	"github.com/pion/rtcp"
	"github.com/pion/rtp"
)

//rtp  in--->plugin1--->plugin2--->out
//rtcp in<---plugin1<---plugin2<---out
//TODO: maybe https://www.chromium.org/developers/design-documents/video
type plugin interface {
	ID() string
	Init(...interface{})
	PushRTP(*rtp.Packet) error
	PushRTCP(rtcp.Packet) error
	Stop()
}
