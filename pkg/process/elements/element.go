package elements

import (
	"errors"

	"github.com/pion/ion/pkg/process/samples"
)

var (
	// ErrAttachNotSupported returned when attaching elments is not supported
	ErrAttachNotSupported = errors.New("attach not supported")
)

// Element interface
type Element interface {
	Type() string
	Write(*samples.Sample) error
	Attach(Element) error
	Read() <-chan *samples.Sample
	Close()
}
