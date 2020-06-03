package samples

// Types for samples
const (
	TypeOpus = 1
	TypeVP8  = 2
	TypeVP9  = 3
	TypeH264 = 4
)

// Sample of audio or video
type Sample struct {
	Type           int
	Timestamp      uint32
	SequenceNumber uint16
	Properties     map[string]interface{}
	Payload        []byte
}
