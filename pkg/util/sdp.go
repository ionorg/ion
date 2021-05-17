package util

import (
	"strings"

	log "github.com/pion/ion-log"
	"github.com/pion/ion/proto/ion"
	"github.com/pixelbender/go-sdp/sdp"
)

//ParseSDP .
func ParseSDP(sdpstr string) ([]*ion.Stream, error) {
	sess, err := sdp.ParseString(sdpstr)

	if err != nil {
		log.Errorf("sdp.Parse erro %v", err)
		return nil, err
	}

	streams := make(map[string]*ion.Stream)
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
			trackLabel := strs[1]

			track := &ion.Track{
				Kind:  m.Type,
				Id:    trackID,
				Label: trackLabel,
			}

			simulcast := make(map[string]string)
			for _, attr := range m.Attributes {
				//fmt.Printf("attr name = %v, value = %v\n", attr.Name, attr.Value)

				if attr.Name == "rid" {
					strs := strings.Split(attr.Value, " ")
					rid := strs[0]
					dir := strs[1]
					//fmt.Printf("rid: rid = %v, dir = %v\n", rid, dir)
					simulcast[rid] = dir
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
			track.Simulcast = simulcast

			if stream, ok := streams[streamID]; ok {
				stream.Tracks = append(stream.Tracks, track)
			} else {
				stream = &ion.Stream{
					Id: streamID,
				}
				stream.Tracks = append(stream.Tracks, track)
				streams[streamID] = stream
			}
		}
	}

	var list []*ion.Stream
	for _, stream := range streams {
		//fmt.Printf("%v\n", stream.String())
		list = append(list, stream)
	}

	return list, nil
}
