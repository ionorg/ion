package elements

import (
	"errors"

	"github.com/pion/ion/pkg/node/avp/process/samplebuilder"
	"github.com/pion/ion/pkg/proto"
)

const (
	// TypeWebmSaver type for webmsaver
	TypeWebmSaver = "WebmSaver"
)

// Element interface
type Element interface {
	Write(*samplebuilder.Sample) error
	Read() <-chan *samplebuilder.Sample
	Close()
}

// GetElement returns an element if valid
func GetElement(msg proto.ElementInfo) (Element, error) {
	switch msg.Type {
	case TypeWebmSaver:
		return NewWebmSaver(msg.MID), nil
	}

	return nil, errors.New("element not found")
}
