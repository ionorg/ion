package sfu

import (
	"strings"

	"github.com/notedit/sdp"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	transport "github.com/pion/ion/pkg/rtc/transport"
)

func getSubPTForTrack(track proto.TrackInfo, sdpObj *sdp.SDPInfo) uint8 {

	medias := sdpObj.GetMedias()
	log.Infof("Medias are %v", medias)

	transform := transport.PaylaodTransformMap()

	for _, m := range medias {
		for _, codec := range m.GetCodecs() {
			log.Infof("Codes are %v", codec)
			pt := codec.GetType()
			// 	If offer contains pub PT, use that
			if track.Payload == pt {
				return uint8(track.Payload)
			}
			// Otherwise look for first supported pt that can be transformed from pub
			if strings.EqualFold(codec.GetCodec(), track.Codec) {
				for _, k := range transform[uint8(track.Payload)] {
					if uint8(pt) == k {
						return k
					}
				}
			}
		}
	}

	return 0
}
