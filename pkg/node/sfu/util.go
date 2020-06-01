package sfu

import (
	"strings"

	"github.com/notedit/sdp"
	"github.com/pion/ion/pkg/log"
	"github.com/pion/ion/pkg/proto"
	transport "github.com/pion/ion/pkg/rtc/transport"
	"github.com/pion/webrtc/v2"
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

func getPubPTForTrack(videoCodec string, track *sdp.TrackInfo, sdpObj *sdp.SDPInfo) (pt uint8, codecName string) {
	media := sdpObj.GetMedia(track.GetMedia())
	codecs := media.GetCodecs()

	for payload, codec := range codecs {
		log.Infof("Codec type %v", codec.GetType())
		if track.GetMedia() == "audio" {
			codecName = strings.ToUpper(codec.GetCodec())
			if strings.EqualFold(codec.GetCodec(), webrtc.Opus) {
				pt = uint8(payload)
				break
			}
		} else if track.GetMedia() == "video" {
			codecName = strings.ToUpper(codec.GetCodec())
			//skip 126 for pub, chrome sub will decode fail when H264 playload type is 126
			if codecName == "H264" && payload == 126 {
				continue
			}
			if codecName == videoCodec {
				pt = uint8(payload)
				break
			}
		}
	}
	return
}
