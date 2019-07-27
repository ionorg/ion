package log

import (
	"os"

	"github.com/pion/ion/conf"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

const (
	// timeFormat = "2006-01-02T15:04:05.999Z07:00"
	timeFormat = "2006-01-02 15:04:05.999"
)

func init() {
	zerolog.TimeFieldFormat = timeFormat
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: timeFormat}
	log = zerolog.New(output).Level(getLogLevel()).With().Timestamp().Logger()
}

func getLogLevel() zerolog.Level {
	switch conf.Log.Level {
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	}
	return zerolog.GlobalLevel()
}

func Infof(format string, v ...interface{}) {
	log.Info().Msgf(format, v...)
}

func Debugf(format string, v ...interface{}) {
	log.Debug().Msgf(format, v...)
}

func Warnf(format string, v ...interface{}) {
	log.Warn().Msgf(format, v...)
}

func Errorf(format string, v ...interface{}) {
	log.Error().Msgf(format, v...)
}

func Panicf(format string, v ...interface{}) {
	log.Panic().Msgf(format, v...)
}
