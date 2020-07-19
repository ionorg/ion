package log

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

const (
	timeFormat = "2006-01-02 15:04:05.000"
)

// Config defines parameters for the logger
type Config struct {
	Level string `mapstructure:"level"`
}

// Init initializes the package logger.
// Supported levels are: ["debug", "info", "warn", "error"]
func Init(level string) {
	l := zerolog.GlobalLevel()
	switch level {
	case "trace":
		l = zerolog.TraceLevel
	case "debug":
		l = zerolog.DebugLevel
	case "info":
		l = zerolog.InfoLevel
	case "warn":
		l = zerolog.WarnLevel
	case "error":
		l = zerolog.ErrorLevel
	}
	zerolog.TimeFieldFormat = timeFormat
	output := zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false, TimeFormat: timeFormat}
	output.FormatTimestamp = func(i interface{}) string {
		return "[" + i.(string) + "]"
	}
	output.FormatLevel = func(i interface{}) string {
		return strings.ToUpper(fmt.Sprintf("[%-3s]", i))
	}
	output.FormatMessage = func(i interface{}) string {
		caller, file, line, _ := runtime.Caller(9)
		fileName := filepath.Base(file)
		funcName := strings.TrimPrefix(filepath.Ext((runtime.FuncForPC(caller).Name())), ".")
		return fmt.Sprintf("[%d][%s][%s] => %s", line, fileName, funcName, i)
	}
	log = zerolog.New(output).Level(l).With().Timestamp().Logger()
}

// Infof logs a formatted info level log to the console
func Infof(format string, v ...interface{}) {
	log.Info().Msgf(format, v...)
}

// Tracef logs a formatted debug level log to the console
func Tracef(format string, v ...interface{}) {
	log.Trace().Msgf(format, v...)
}

// Debugf logs a formatted debug level log to the console
func Debugf(format string, v ...interface{}) {
	log.Debug().Msgf(format, v...)
}

// Warnf logs a formatted warn level log to the console
func Warnf(format string, v ...interface{}) {
	log.Warn().Msgf(format, v...)
}

// Errorf logs a formatted error level log to the console
func Errorf(format string, v ...interface{}) {
	log.Error().Msgf(format, v...)
}

// Panicf logs a formatted panic level log to the console.
// The panic() function is called, which stops the ordinary flow of a goroutine.
func Panicf(format string, v ...interface{}) {
	log.Panic().Msgf(format, v...)
}
