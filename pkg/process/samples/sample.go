package samples

// Types for samples
const (
	TypeOpus  = 111
	TypeVP8   = 96
	TypeVP9   = 98
	TypeH264  = 102
	TypeRGB24 = 200
)

// Sample of audio or video
type Sample struct {
	Type           int
	Timestamp      uint32
	SequenceNumber uint16
	Properties     map[string]interface{}
	Payload        []byte
}
