package dtls

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"

	"golang.org/x/xerrors"
)

// Typed errors
var (
	ErrConnClosed = &FatalError{errors.New("conn is closed")}

	errDeadlineExceeded = &TimeoutError{xerrors.Errorf("read/write timeout: %w", context.DeadlineExceeded)}

	errBufferTooSmall               = &TemporaryError{errors.New("buffer is too small")}
	errContextUnsupported           = &TemporaryError{errors.New("context is not supported for ExportKeyingMaterial")}
	errDTLSPacketInvalidLength      = &TemporaryError{errors.New("packet is too short")}
	errHandshakeInProgress          = &TemporaryError{errors.New("handshake is in progress")}
	errInvalidContentType           = &TemporaryError{errors.New("invalid content type")}
	errInvalidMAC                   = &TemporaryError{errors.New("invalid mac")}
	errInvalidPacketLength          = &TemporaryError{errors.New("packet length and declared length do not match")}
	errReservedExportKeyingMaterial = &TemporaryError{errors.New("ExportKeyingMaterial can not be used with a reserved label")}

	errCertificateVerifyNoCertificate   = &FatalError{errors.New("client sent certificate verify but we have no certificate to verify")}
	errCipherSuiteNoIntersection        = &FatalError{errors.New("client+server do not support any shared cipher suites")}
	errCipherSuiteUnset                 = &FatalError{errors.New("server hello can not be created without a cipher suite")}
	errClientCertificateNotVerified     = &FatalError{errors.New("client sent certificate but did not verify it")}
	errClientCertificateRequired        = &FatalError{errors.New("server required client verification, but got none")}
	errClientNoMatchingSRTPProfile      = &FatalError{errors.New("server responded with SRTP Profile we do not support")}
	errClientRequiredButNoServerEMS     = &FatalError{errors.New("client required Extended Master Secret extension, but server does not support it")}
	errCompressionMethodUnset           = &FatalError{errors.New("server hello can not be created without a compression method")}
	errCookieMismatch                   = &FatalError{errors.New("client+server cookie does not match")}
	errCookieTooLong                    = &FatalError{errors.New("cookie must not be longer then 255 bytes")}
	errIdentityNoPSK                    = &FatalError{errors.New("PSK Identity Hint provided but PSK is nil")}
	errInvalidCertificate               = &FatalError{errors.New("no certificate provided")}
	errInvalidCipherSpec                = &FatalError{errors.New("cipher spec invalid")}
	errInvalidCipherSuite               = &FatalError{errors.New("invalid or unknown cipher suite")}
	errInvalidClientKeyExchange         = &FatalError{errors.New("unable to determine if ClientKeyExchange is a public key or PSK Identity")}
	errInvalidCompressionMethod         = &FatalError{errors.New("invalid or unknown compression method")}
	errInvalidECDSASignature            = &FatalError{errors.New("ECDSA signature contained zero or negative values")}
	errInvalidEllipticCurveType         = &FatalError{errors.New("invalid or unknown elliptic curve type")}
	errInvalidExtensionType             = &FatalError{errors.New("invalid extension type")}
	errInvalidHashAlgorithm             = &FatalError{errors.New("invalid hash algorithm")}
	errInvalidNamedCurve                = &FatalError{errors.New("invalid named curve")}
	errInvalidPrivateKey                = &FatalError{errors.New("invalid private key type")}
	errInvalidSNIFormat                 = &FatalError{errors.New("invalid server name format")}
	errInvalidSignatureAlgorithm        = &FatalError{errors.New("invalid signature algorithm")}
	errKeySignatureMismatch             = &FatalError{errors.New("expected and actual key signature do not match")}
	errNilNextConn                      = &FatalError{errors.New("Conn can not be created with a nil nextConn")}
	errNoAvailableCipherSuites          = &FatalError{errors.New("connection can not be created, no CipherSuites satisfy this Config")}
	errNoAvailableSignatureSchemes      = &FatalError{errors.New("connection can not be created, no SignatureScheme satisfy this Config")}
	errNoCertificates                   = &FatalError{errors.New("no certificates configured")}
	errNoConfigProvided                 = &FatalError{errors.New("no config provided")}
	errNoSupportedEllipticCurves        = &FatalError{errors.New("client requested zero or more elliptic curves that are not supported by the server")}
	errUnsupportedProtocolVersion       = &FatalError{errors.New("unsupported protocol version")}
	errPSKAndCertificate                = &FatalError{errors.New("Certificate and PSK provided")} // nolint:stylecheck
	errPSKAndIdentityMustBeSetForClient = &FatalError{errors.New("PSK and PSK Identity Hint must both be set for client")}
	errRequestedButNoSRTPExtension      = &FatalError{errors.New("SRTP support was requested but server did not respond with use_srtp extension")}
	errServerMustHaveCertificate        = &FatalError{errors.New("Certificate is mandatory for server")} // nolint:stylecheck
	errServerNoMatchingSRTPProfile      = &FatalError{errors.New("client requested SRTP but we have no matching profiles")}
	errServerRequiredButNoClientEMS     = &FatalError{errors.New("server requires the Extended Master Secret extension, but the client does not support it")}
	errVerifyDataMismatch               = &FatalError{errors.New("expected and actual verify data does not match")}

	errHandshakeMessageUnset             = &InternalError{errors.New("handshake message unset, unable to marshal")}
	errInvalidFlight                     = &InternalError{errors.New("invalid flight number")}
	errKeySignatureGenerateUnimplemented = &InternalError{errors.New("unable to generate key signature, unimplemented")}
	errKeySignatureVerifyUnimplemented   = &InternalError{errors.New("unable to verify key signature, unimplemented")}
	errLengthMismatch                    = &InternalError{errors.New("data length and declared length do not match")}
	errNotEnoughRoomForNonce             = &InternalError{errors.New("buffer not long enough to contain nonce")}
	errNotImplemented                    = &InternalError{errors.New("feature has not been implemented yet")}
	errSequenceNumberOverflow            = &InternalError{errors.New("sequence number overflow")}
	errUnableToMarshalFragmented         = &InternalError{errors.New("unable to marshal fragmented handshakes")}
)

// FatalError indicates that the DTLS connection is no longer available.
// It is mainly caused by wrong configuration of server or client.
type FatalError struct {
	Err error
}

// InternalError indicates and internal error caused by the implementation, and the DTLS connection is no longer available.
// It is mainly caused by bugs or tried to use unimplemented features.
type InternalError struct {
	Err error
}

// TemporaryError indicates that the DTLS connection is still available, but the request was failed temporary.
type TemporaryError struct {
	Err error
}

// TimeoutError indicates that the request was timed out.
type TimeoutError struct {
	Err error
}

// HandshakeError indicates that the handshake failed.
type HandshakeError struct {
	Err error
}

// invalidCipherSuite indicates an attempt at using an unsupported cipher suite.
type invalidCipherSuite struct {
	id CipherSuiteID
}

func (e *invalidCipherSuite) Error() string {
	return fmt.Sprintf("CipherSuite with id(%d) is not valid", e.id)
}

func (e *invalidCipherSuite) Is(err error) bool {
	if other, ok := err.(*invalidCipherSuite); ok {
		return e.id == other.id
	}
	return false
}

// Timeout implements net.Error.Timeout()
func (*FatalError) Timeout() bool { return false }

// Temporary implements net.Error.Temporary()
func (*FatalError) Temporary() bool { return false }

// Unwrap implements Go1.13 error unwrapper.
func (e *FatalError) Unwrap() error { return e.Err }

func (e *FatalError) Error() string { return fmt.Sprintf("dtls fatal: %v", e.Err) }

// Timeout implements net.Error.Timeout()
func (*InternalError) Timeout() bool { return false }

// Temporary implements net.Error.Temporary()
func (*InternalError) Temporary() bool { return false }

// Unwrap implements Go1.13 error unwrapper.
func (e *InternalError) Unwrap() error { return e.Err }

func (e *InternalError) Error() string { return fmt.Sprintf("dtls internal: %v", e.Err) }

// Timeout implements net.Error.Timeout()
func (*TemporaryError) Timeout() bool { return false }

// Temporary implements net.Error.Temporary()
func (*TemporaryError) Temporary() bool { return true }

// Unwrap implements Go1.13 error unwrapper.
func (e *TemporaryError) Unwrap() error { return e.Err }

func (e *TemporaryError) Error() string { return fmt.Sprintf("dtls temporary: %v", e.Err) }

// Timeout implements net.Error.Timeout()
func (*TimeoutError) Timeout() bool { return true }

// Temporary implements net.Error.Temporary()
func (*TimeoutError) Temporary() bool { return true }

// Unwrap implements Go1.13 error unwrapper.
func (e *TimeoutError) Unwrap() error { return e.Err }

func (e *TimeoutError) Error() string { return fmt.Sprintf("dtls timeout: %v", e.Err) }

// Timeout implements net.Error.Timeout()
func (e *HandshakeError) Timeout() bool {
	if netErr, ok := e.Err.(net.Error); ok {
		return netErr.Timeout()
	}
	return false
}

// Temporary implements net.Error.Temporary()
func (e *HandshakeError) Temporary() bool {
	if netErr, ok := e.Err.(net.Error); ok {
		return netErr.Temporary()
	}
	return false
}

// Unwrap implements Go1.13 error unwrapper.
func (e *HandshakeError) Unwrap() error { return e.Err }

func (e *HandshakeError) Error() string { return fmt.Sprintf("handshake error: %v", e.Err) }

// errAlert wraps DTLS alert notification as an error
type errAlert struct {
	*alert
}

func (e *errAlert) Error() string {
	return fmt.Sprintf("alert: %s", e.alert.String())
}

func (e *errAlert) IsFatalOrCloseNotify() bool {
	return e.alertLevel == alertLevelFatal || e.alertDescription == alertCloseNotify
}

func (e *errAlert) Is(err error) bool {
	if other, ok := err.(*errAlert); ok {
		return e.alertLevel == other.alertLevel && e.alertDescription == other.alertDescription
	}
	return false
}

// netError translates an error from underlying Conn to corresponding net.Error.
func netError(err error) error {
	switch err {
	case io.EOF, context.Canceled, context.DeadlineExceeded:
		// Return io.EOF and context errors as is.
		return err
	}
	switch e := err.(type) {
	case (*net.OpError):
		if se, ok := e.Err.(*os.SyscallError); ok {
			if se.Timeout() {
				return &TimeoutError{err}
			}
			if isOpErrorTemporary(se) {
				return &TemporaryError{err}
			}
		}
	case (net.Error):
		return err
	}
	return &FatalError{err}
}
