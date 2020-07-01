package elements

import "errors"

// Types for samples
const (
	TypeMetadata = 100
	TypeBinary   = 101
	TypeRGB24    = 102
	TypeWebM     = 103
)

var (
	// ErrAttachNotSupported returned when attaching elements is not supported
	ErrAttachNotSupported = errors.New("attach not supported")
	// ErrElementAlreadyAttached returned when attaching an element that is already attached
	ErrElementAlreadyAttached = errors.New("element already attached")
)
