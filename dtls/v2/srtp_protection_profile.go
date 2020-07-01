package dtls

// SRTPProtectionProfile defines the parameters and options that are in effect for the SRTP processing
// https://tools.ietf.org/html/rfc5764#section-4.1.2
type SRTPProtectionProfile uint16

const (
	SRTP_AES128_CM_HMAC_SHA1_80 SRTPProtectionProfile = 0x0001 // nolint
)

var srtpProtectionProfiles = map[SRTPProtectionProfile]bool{
	SRTP_AES128_CM_HMAC_SHA1_80: true,
}
