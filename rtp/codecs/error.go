package codecs

import "errors"

var (
	errShortPacket  = errors.New("packet is not large enough")
	errNilPacket    = errors.New("invalid nil packet")
	errTooManyPDiff = errors.New("too many PDiff")
)
