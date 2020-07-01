package srtp

import (
	"crypto/cipher"
	"crypto/subtle"
	"encoding/binary"
	"fmt"

	"github.com/sssgun/ion/rtcp"
)

const maxSRTCPIndex = 0x7FFFFFFF

func (c *Context) decryptRTCP(dst, encrypted []byte) ([]byte, error) {
	out := allocateIfMismatch(dst, encrypted)

	tailOffset := len(encrypted) - (authTagSize + srtcpIndexSize)
	out = out[0:tailOffset]

	isEncrypted := encrypted[tailOffset] >> 7
	if isEncrypted == 0 {
		return out, nil
	}

	srtcpIndexBuffer := encrypted[tailOffset : tailOffset+srtcpIndexSize]

	index := binary.BigEndian.Uint32(srtcpIndexBuffer) &^ (1 << 31)
	ssrc := binary.BigEndian.Uint32(encrypted[4:])

	s := c.getSRTCPSSRCState(ssrc)

	markAsValid, ok := s.replayDetector.Check(uint64(index))
	if !ok {
		return nil, errDuplicated
	}

	actualTag := encrypted[len(encrypted)-authTagSize:]
	expectedTag, err := c.generateSrtcpAuthTag(encrypted[:len(encrypted)-authTagSize])
	if err != nil {
		return nil, err
	}

	if subtle.ConstantTimeCompare(actualTag, expectedTag) != 1 {
		return nil, fmt.Errorf("failed to verify auth tag")
	}
	markAsValid()

	stream := cipher.NewCTR(c.srtcpBlock, c.generateCounter(uint16(index&0xffff), index>>16, ssrc, c.srtcpSessionSalt))
	stream.XORKeyStream(out[8:], out[8:])

	return out, nil
}

// DecryptRTCP decrypts a buffer that contains a RTCP packet
func (c *Context) DecryptRTCP(dst, encrypted []byte, header *rtcp.Header) ([]byte, error) {
	if header == nil {
		header = &rtcp.Header{}
	}

	if err := header.Unmarshal(encrypted); err != nil {
		return nil, err
	}

	return c.decryptRTCP(dst, encrypted)
}

func (c *Context) encryptRTCP(dst, decrypted []byte) ([]byte, error) {
	out := allocateIfMismatch(dst, decrypted)
	ssrc := binary.BigEndian.Uint32(out[4:])
	s := c.getSRTCPSSRCState(ssrc)

	// We roll over early because MSB is used for marking as encrypted
	s.srtcpIndex++
	if s.srtcpIndex >= maxSRTCPIndex {
		s.srtcpIndex = 0
	}

	// Encrypt everything after header
	stream := cipher.NewCTR(c.srtcpBlock, c.generateCounter(uint16(s.srtcpIndex&0xffff), s.srtcpIndex>>16, ssrc, c.srtcpSessionSalt))
	stream.XORKeyStream(out[8:], out[8:])

	// Add SRTCP Index and set Encryption bit
	out = append(out, make([]byte, 4)...)
	binary.BigEndian.PutUint32(out[len(out)-4:], s.srtcpIndex)
	out[len(out)-4] |= 0x80

	authTag, err := c.generateSrtcpAuthTag(out)
	if err != nil {
		return nil, err
	}
	return append(out, authTag...), nil
}

// EncryptRTCP Encrypts a RTCP packet
func (c *Context) EncryptRTCP(dst, decrypted []byte, header *rtcp.Header) ([]byte, error) {
	if header == nil {
		header = &rtcp.Header{}
	}

	if err := header.Unmarshal(decrypted); err != nil {
		return nil, err
	}

	return c.encryptRTCP(dst, decrypted)
}
