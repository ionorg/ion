package node

import (
	logger "github.com/cloudwebrtc/nats-protoo/logger"
	"github.com/pion/ion/pkg/log"
	"github.com/sirupsen/logrus"
)

// Init init loggers.
func Init() {
	logger.Init("debug")
	log.Init("debug")
	logrus.SetLevel(logrus.DebugLevel)
}
