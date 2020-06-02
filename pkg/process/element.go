package process

import (
	"github.com/pion/ion/pkg/process/samples"
)

// Element interface
type Element interface {
	Type() string
	Write(*samples.Sample) error
	Attach(Element) error
	Read() <-chan *samples.Sample
	Close()
}
