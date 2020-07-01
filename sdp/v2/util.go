package sdp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sort"
	"strconv"
	"strings"

	"time"
)

const (
	attributeKey = "a="
)

// ConnectionRole indicates which of the end points should initiate the connection establishment
type ConnectionRole int

const (
	// ConnectionRoleActive indicates the endpoint will initiate an outgoing connection.
	ConnectionRoleActive ConnectionRole = iota + 1

	// ConnectionRolePassive indicates the endpoint will accept an incoming connection.
	ConnectionRolePassive

	// ConnectionRoleActpass indicates the endpoint is willing to accept an incoming connection or to initiate an outgoing connection.
	ConnectionRoleActpass

	// ConnectionRoleHoldconn indicates the endpoint does not want the connection to be established for the time being.
	ConnectionRoleHoldconn
)

func (t ConnectionRole) String() string {
	switch t {
	case ConnectionRoleActive:
		return "active"
	case ConnectionRolePassive:
		return "passive"
	case ConnectionRoleActpass:
		return "actpass"
	case ConnectionRoleHoldconn:
		return "holdconn"
	default:
		return "Unknown"
	}
}

func newSessionID() uint64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return uint64(r.Uint32()*2) >> 2
}

// Codec represents a codec
type Codec struct {
	PayloadType        uint8
	Name               string
	ClockRate          uint32
	EncodingParameters string
	Fmtp               string
	RTCPFeedback       []string
}

const (
	unknown = iota
)

func (c Codec) String() string {
	return fmt.Sprintf("%d %s/%d/%s (%s) [%s]", c.PayloadType, c.Name, c.ClockRate, c.EncodingParameters, c.Fmtp, strings.Join(c.RTCPFeedback, ", "))
}

func parseRtpmap(rtpmap string) (Codec, error) {
	var codec Codec
	parsingFailed := errors.New("could not extract codec from rtpmap")

	// a=rtpmap:<payload type> <encoding name>/<clock rate>[/<encoding parameters>]
	split := strings.Split(rtpmap, " ")
	if len(split) != 2 {
		return codec, parsingFailed
	}

	ptSplit := strings.Split(split[0], ":")
	if len(ptSplit) != 2 {
		return codec, parsingFailed
	}

	ptInt, err := strconv.Atoi(ptSplit[1])
	if err != nil {
		return codec, parsingFailed
	}

	codec.PayloadType = uint8(ptInt)

	split = strings.Split(split[1], "/")
	codec.Name = split[0]
	parts := len(split)
	if parts > 1 {
		rate, err := strconv.Atoi(split[1])
		if err != nil {
			return codec, parsingFailed
		}
		codec.ClockRate = uint32(rate)
	}
	if parts > 2 {
		codec.EncodingParameters = split[2]
	}

	return codec, nil
}

func parseFmtp(fmtp string) (Codec, error) {
	var codec Codec
	parsingFailed := errors.New("could not extract codec from fmtp")

	// a=fmtp:<format> <format specific parameters>
	split := strings.Split(fmtp, " ")
	if len(split) != 2 {
		return codec, parsingFailed
	}

	formatParams := split[1]

	split = strings.Split(split[0], ":")
	if len(split) != 2 {
		return codec, parsingFailed
	}

	ptInt, err := strconv.Atoi(split[1])
	if err != nil {
		return codec, parsingFailed
	}

	codec.PayloadType = uint8(ptInt)
	codec.Fmtp = formatParams

	return codec, nil
}

func parseRtcpFb(rtcpFb string) (Codec, error) {
	var codec Codec
	parsingFailed := errors.New("could not extract codec from rtcp-fb")

	// a=ftcp-fb:<payload type> <RTCP feedback type> [<RTCP feedback parameter>]
	split := strings.SplitN(rtcpFb, " ", 2)
	if len(split) != 2 {
		return codec, parsingFailed
	}

	ptSplit := strings.Split(split[0], ":")
	if len(ptSplit) != 2 {
		return codec, parsingFailed
	}

	ptInt, err := strconv.Atoi(ptSplit[1])
	if err != nil {
		return codec, parsingFailed
	}

	codec.PayloadType = uint8(ptInt)
	codec.RTCPFeedback = append(codec.RTCPFeedback, split[1])

	return codec, nil
}

func mergeCodecs(codec Codec, codecs map[uint8]Codec) {
	savedCodec := codecs[codec.PayloadType]

	if savedCodec.PayloadType == 0 {
		savedCodec.PayloadType = codec.PayloadType
	}
	if savedCodec.Name == "" {
		savedCodec.Name = codec.Name
	}
	if savedCodec.ClockRate == 0 {
		savedCodec.ClockRate = codec.ClockRate
	}
	if savedCodec.EncodingParameters == "" {
		savedCodec.EncodingParameters = codec.EncodingParameters
	}
	if savedCodec.Fmtp == "" {
		savedCodec.Fmtp = codec.Fmtp
	}
	savedCodec.RTCPFeedback = append(savedCodec.RTCPFeedback, codec.RTCPFeedback...)

	codecs[savedCodec.PayloadType] = savedCodec
}

func (s *SessionDescription) buildCodecMap() map[uint8]Codec {
	codecs := make(map[uint8]Codec)

	for _, m := range s.MediaDescriptions {
		for _, a := range m.Attributes {
			attr := *a.String()
			if strings.HasPrefix(attr, "rtpmap:") {
				codec, err := parseRtpmap(attr)
				if err == nil {
					mergeCodecs(codec, codecs)
				}
			} else if strings.HasPrefix(attr, "fmtp:") {
				codec, err := parseFmtp(attr)
				if err == nil {
					mergeCodecs(codec, codecs)
				}
			} else if strings.HasPrefix(attr, "rtcp-fb:") {
				codec, err := parseRtcpFb(attr)
				if err == nil {
					mergeCodecs(codec, codecs)
				}
			}
		}
	}

	return codecs
}

func equivalentFmtp(want, got string) bool {
	wantSplit := strings.Split(want, ";")
	gotSplit := strings.Split(got, ";")

	if len(wantSplit) != len(gotSplit) {
		return false
	}

	sort.Strings(wantSplit)
	sort.Strings(gotSplit)

	for i, wantPart := range wantSplit {
		wantPart = strings.TrimSpace(wantPart)
		gotPart := strings.TrimSpace(gotSplit[i])
		if gotPart != wantPart {
			return false
		}
	}

	return true
}

func codecsMatch(wanted, got Codec) bool {
	if wanted.Name != "" && !strings.EqualFold(wanted.Name, got.Name) {
		return false
	}
	if wanted.ClockRate != 0 && wanted.ClockRate != got.ClockRate {
		return false
	}
	if wanted.EncodingParameters != "" && wanted.EncodingParameters != got.EncodingParameters {
		return false
	}
	if wanted.Fmtp != "" && !equivalentFmtp(wanted.Fmtp, got.Fmtp) {
		return false
	}

	return true
}

// GetCodecForPayloadType scans the SessionDescription for the given payload type and returns the codec
func (s *SessionDescription) GetCodecForPayloadType(payloadType uint8) (Codec, error) {
	codecs := s.buildCodecMap()

	codec, ok := codecs[payloadType]
	if ok {
		return codec, nil
	}

	return codec, errors.New("payload type not found")
}

// GetPayloadTypeForCodec scans the SessionDescription for a codec that matches the provided codec
// as closely as possible and returns its payload type
func (s *SessionDescription) GetPayloadTypeForCodec(wanted Codec) (uint8, error) {
	codecs := s.buildCodecMap()

	for payloadType, codec := range codecs {
		if codecsMatch(wanted, codec) {
			return payloadType, nil
		}
	}

	return 0, errors.New("codec not found")
}

type lexer struct {
	desc  *SessionDescription
	input *bufio.Reader
}

type stateFn func(*lexer) (stateFn, error)

func readType(input *bufio.Reader) (string, error) {
	for {
		b, err := input.ReadByte()
		if err != nil {
			return "", err
		}
		if b == '\n' || b == '\r' {
			continue
		}
		if err = input.UnreadByte(); err != nil {
			return "", err
		}

		key, err := input.ReadString('=')
		if err != nil {
			return key, err
		}

		switch len(key) {
		case 2:
			return key, nil
		default:
			return key, fmt.Errorf("SyntaxError: %v", strconv.Quote(key))
		}
	}
}

func readValue(input *bufio.Reader) (string, error) {
	lineBytes, _, err := input.ReadLine()
	line := string(lineBytes)
	if err != nil && err != io.EOF {
		return line, err
	}

	if len(line) == 0 {
		return line, io.EOF
	}

	return line, nil
}

func indexOf(element string, data []string) int {
	for k, v := range data {
		if element == v {
			return k
		}
	}
	return -1
}

func keyValueBuild(key string, value *string) string {
	if value != nil {
		return key + *value + "\r\n"
	}
	return ""
}
