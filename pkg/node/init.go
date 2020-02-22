package node

import (
	logger "github.com/cloudwebrtc/nats-protoo/logger"
)

// Init init loggers.
func Init() {
	logger.Init("debug")
}
