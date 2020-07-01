// Package srtp implements Secure Real-time Transport Protocol
package srtp

import (
	"crypto/cipher"
	"crypto/subtle"
	"fmt"

	"github.com/sssgun/ion/rtp"
)

func (c *Context) decryptRTP(dst, ciphertext []byte, header *rtp.Header) ([]byte, error) {
	s := c.getSRTPSSRCState(header.SSRC)

	markAsValid, ok := s.replayDetector.Check(uint64(header.SequenceNumber))
	if !ok {
		return nil, errDuplicated
	}

	dst = growBufferSize(dst, len(ciphertext)-authTagSize)

	c.updateRolloverCount(header.SequenceNumber, s)

	// Split the auth tag and the cipher text into two parts.
	actualTag := ciphertext[len(ciphertext)-authTagSize:]
	ciphertext = ciphertext[:len(ciphertext)-authTagSize]

	// Generate the auth tag we expect to see from the ciphertext.
	expectedTag, err := c.generateSrtpAuthTag(ciphertext, s.rolloverCounter)
	if err != nil {
		return nil, err
	}

	// See if the auth tag actually matches.
	// We use a constant time comparison to prevent timing attacks.
	if subtle.ConstantTimeCompare(actualTag, expectedTag) != 1 {
		return nil, fmt.Errorf("failed to verify auth tag")
	}
	markAsValid()

	// Write the plaintext header to the destination buffer.
	copy(dst, ciphertext[:header.PayloadOffset])

	// Decrypt the ciphertext for the payload.
	counter := c.generateCounter(header.SequenceNumber, s.rolloverCounter, s.ssrc, c.srtpSessionSalt)
	stream := cipher.NewCTR(c.srtpBlock, counter)
	stream.XORKeyStream(dst[header.PayloadOffset:], ciphertext[header.PayloadOffset:])

	return dst, nil
}

// DecryptRTP decrypts a RTP packet with an encrypted payload
func (c *Context) DecryptRTP(dst, encrypted []byte, header *rtp.Header) ([]byte, error) {
	if header == nil {
		header = &rtp.Header{}
	}

	if err := header.Unmarshal(encrypted); err != nil {
		return nil, err
	}

	return c.decryptRTP(dst, encrypted, header)
}

// EncryptRTP marshals and encrypts an RTP packet, writing to the dst buffer provided.
// If the dst buffer does not have the capacity to hold `len(plaintext) + 10` bytes, a new one will be allocated and returned.
// If a rtp.Header is provided, it will be Unmarshaled using the plaintext.
func (c *Context) EncryptRTP(dst []byte, plaintext []byte, header *rtp.Header) ([]byte, error) {
	if header == nil {
		header = &rtp.Header{}
	}

	err := header.Unmarshal(plaintext)
	if err != nil {
		return nil, err
	}

	return c.encryptRTP(dst, header, plaintext[header.PayloadOffset:])
}

// encryptRTP marshals and encrypts an RTP packet, writing to the dst buffer provided.
// If the dst buffer does not have the capacity to hold `len(plaintext) + 10` bytes, a new one will be allocated and returned.
// Similar to above but faster because it can avoid unmarshaling the header and marshaling the payload.
func (c *Context) encryptRTP(dst []byte, header *rtp.Header, payload []byte) (ciphertext []byte, err error) {
	// Grow the given buffer to fit the output.
	// authTag = 10 bytes
	dst = growBufferSize(dst, header.MarshalSize()+len(payload)+10)

	s := c.getSRTPSSRCState(header.SSRC)
	c.updateRolloverCount(header.SequenceNumber, s)

	// Copy the header unencrypted.
	n, err := header.MarshalTo(dst)
	if err != nil {
		return nil, err
	}

	// Encrypt the payload
	counter := c.generateCounter(header.SequenceNumber, s.rolloverCounter, s.ssrc, c.srtpSessionSalt)
	stream := cipher.NewCTR(c.srtpBlock, counter)
	stream.XORKeyStream(dst[n:], payload)
	n += len(payload)

	// Generate the auth tag.
	authTag, err := c.generateSrtpAuthTag(dst[:n], s.rolloverCounter)
	if err != nil {
		return nil, err
	}

	// Write the auth tag to the dest.
	copy(dst[n:], authTag)

	return dst, nil
}
