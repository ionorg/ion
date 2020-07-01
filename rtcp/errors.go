package rtcp

import "errors"

var (
	errInvalidTotalLost  = errors.New("rtcp: invalid total lost count")
	errInvalidHeader     = errors.New("rtcp: invalid header")
	errEmptyCompound     = errors.New("rtcp: empty compound packet")
	errBadFirstPacket    = errors.New("rtcp: first packet in compound must be SR or RR")
	errMissingCNAME      = errors.New("rtcp: compound missing SourceDescription with CNAME")
	errPacketBeforeCNAME = errors.New("rtcp: feedback packet seen before CNAME")
	errTooManyReports    = errors.New("rtcp: too many reports")
	errTooManyChunks     = errors.New("rtcp: too many chunks")
	errTooManySources    = errors.New("rtcp: too many sources")
	errPacketTooShort    = errors.New("rtcp: packet too short")
	errWrongType         = errors.New("rtcp: wrong packet type")
	errSDESTextTooLong   = errors.New("rtcp: sdes must be < 255 octets long")
	errSDESMissingType   = errors.New("rtcp: sdes item missing type")
	errReasonTooLong     = errors.New("rtcp: reason must be < 255 octets long")
	errBadVersion        = errors.New("rtcp: invalid packet version")
)
