package rtp

import (
	"errors"
)

var errHeaderSizeInsufficient = errors.New("RTP header size insufficient")
var errHeaderSizeInsufficientForExtension = errors.New("RTP header size insufficient for extension")
