package log

import (
	"os"
	"time"

	"github.com/pion/sfu/conf"
	"github.com/rs/zerolog"
)

var log zerolog.Logger

// var log zerolog.ConsoleWriter

// Level defines log levels.
type Level uint8

const (
	// DebugLevel defines debug log level.
	DebugLevel Level = iota
	// InfoLevel defines info log level.
	InfoLevel
	// WarnLevel defines warn log level.
	WarnLevel
	// ErrorLevel defines error log level.
	ErrorLevel
	// FatalLevel defines fatal log level.
	FatalLevel
	// PanicLevel defines panic log level.
	PanicLevel
	// NoLevel defines an absent log level.
	NoLevel
	// Disabled disables the logger.
	Disabled
)

func init() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	// zerolog.TimestampFunc = func() time.Time {
	// return time.Date(2008, 1, 8, 17, 5, 05, 0, time.UTC)
	// }
	// zerolog.TimeFieldFormat = "2006-01-02 15:04:05"
	// log = zerolog.New(os.Stdout).With().Timestamp().Logger()
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log = zerolog.New(output).With().Timestamp().Logger()
	switch conf.Cfg.Log.Level {
	case "debug":
		SetLevel(DebugLevel)
	case "info":
		SetLevel(InfoLevel)
	case "warn":
		SetLevel(WarnLevel)
	case "error":
		SetLevel(ErrorLevel)
	}

}

func SetLevel(l Level) {
	zerolog.SetGlobalLevel(zerolog.Level(l))
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

// func Info(msg string) {
// log.Info().Msg(msg)
// }

// func Debug(msg string) {
// log.Debug().Msg(msg)
// }

// func Warn(msg string) {
// log.Warn().Msg(msg)
// }

// func Error(msg string) {
// log.Error().Msg(msg)
// }

// func Panic(msg string) {
// log.Panic().Msg(msg)
// }
