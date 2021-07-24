package sfu

import (
	"strings"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/proto/rtc"
	"github.com/pixelbender/go-sdp/sdp"
)

//ParseSDP .
func ParseSDP(uid, sdpstr string) ([]*rtc.Stream, error) {
	sess, err := sdp.ParseString(sdpstr)

	if err != nil {
		log.Errorf("sdp.Parse erro %v", err)
		return nil, err
	}

	streams := make(map[string]*rtc.Stream)
	for _, m := range sess.Media {
		//fmt.Printf("type = %v\n", m.Type)

		if m.Type == "audio" || m.Type == "video" {
			msid := m.Attributes.Get("msid")
			//fmt.Printf("msid id = %v\n", msid)

			if msid == "" {
				continue
			}

			strs := strings.Split(msid, " ")
			streamID := strs[0]
			trackID := msid

			track := &rtc.Track{
				Kind: m.Type,
				Id:   trackID,
			}

			/*
				a=rid:f send pt=96;max-width=1280;max-height=720
				a=rid:h send pt=96;max-width=640;max-height=360
				a=rid:q send pt=96;max-width=320;max-height=180
				a=simulcast:send f;h;q
			*/

			stream, ok := streams[streamID]

			if ok {
				stream.Tracks = append(stream.Tracks, track)
			} else {
				stream = &rtc.Stream{
					Uid:  uid,
					Msid: streamID,
				}
				stream.Tracks = append(stream.Tracks, track)
				streams[streamID] = stream
			}

			if m.Type == "video" {
				for _, attr := range m.Attributes {
					if attr.Name == "rid" {
						strs := strings.Split(attr.Value, " ")
						layer := &rtc.Simulcast{}
						layer.Rid = strs[0]
						layer.Direction = strs[1]
						if len(strs) > 2 {
							layer.Parameters = strs[2]
						}
						stream.Simulcast = append(stream.Simulcast, layer)
					}
					/*
						if attr.Name == "simulcast" {
							strs := strings.Split(attr.Value, " ")
							dir := strs[0]
							rids := strs[1]
							fmt.Printf("simulcast: rids = %v, dir = %v\n", rids, dir)
						}
					*/
				}
			}
		}
	}

	var list []*rtc.Stream
	for _, stream := range streams {
		list = append(list, stream)
	}

	return list, nil
}
