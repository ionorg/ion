package elements

import (
	"github.com/pion/ion/pkg/process/samples"
)

// Element interface
type Element interface {
	Write(*samples.Sample) error
	Read() <-chan *samples.Sample
	Close()
}
