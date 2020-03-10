package transport

import (
	"testing"

	"github.com/pion/webrtc/v2"
)

func TestWebRTCTransportOffer(t *testing.T) {
	options := make(map[string]interface{})
	options["codec"] = "h264"
	options["transport-cc"] = ""
	pub := NewWebRTCTransport("pub", options)
	_, err := pub.Offer()
	if err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestWebRTCTransportAnswer(t *testing.T) {
	options := make(map[string]interface{})
	options["codec"] = "h264"
	options["transport-cc"] = ""
	pub := NewWebRTCTransport("pub", options)
	offer, err := pub.Offer()
	if err != nil {
		t.Fatalf("err=%v", err)
	}

	_, err = pub.AddTrack(12345, webrtc.DefaultPayloadTypeH264, "video", "pion")
	if err != nil {
		t.Fatalf("err=%v", err)
	}

	sub := NewWebRTCTransport("sub", options)
	options["subscribe"] = ""
	options["ssrcpt"] = make(map[uint32]uint8)
	for ssrc, track := range pub.GetOutTracks() {
		options["ssrcpt"].(map[uint32]uint8)[ssrc] = track.PayloadType()
	}
	answer, err := sub.Answer(offer, options)
	if err != nil {
		t.Fatalf("err=%v answer=%v", err, answer)
	}
}
