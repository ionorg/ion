package transport

import (
	"testing"

	"github.com/pion/webrtc/v2"
)

func TestWebRTCTransportOffer(t *testing.T) {
	options := RTCOptions{
		Codec:       "h264",
		TransportCC: true,
	}
	pub := NewWebRTCTransport("pub", options)
	_, err := pub.Offer()
	if err != nil {
		t.Fatalf("err=%v", err)
	}
}

func TestWebRTCTransportAnswer(t *testing.T) {
	options := RTCOptions{
		Codec:       "h264",
		TransportCC: true,
	}
	pub := NewWebRTCTransport("pub", options)
	offer, err := pub.Offer()
	if err != nil {
		t.Fatalf("err=%v", err)
	}

	_, err = pub.AddSendTrack(12345, webrtc.DefaultPayloadTypeH264, "video", "pion")
	if err != nil {
		t.Fatalf("err=%v", err)
	}

	sub := NewWebRTCTransport("sub", options)
	options.Subscribe = true
	options.Ssrcpt = make(map[uint32]uint8)
	for ssrc, track := range pub.GetOutTracks() {
		options.Ssrcpt[ssrc] = track.PayloadType()
	}
	answer, err := sub.Answer(offer, options)
	if err != nil {
		t.Fatalf("err=%v answer=%v", err, answer)
	}
}
