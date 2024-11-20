package context

import (
	"github.com/sirupsen/logrus"
)

// LoggerKey is a type for context key
type LoggerKey string

const (
	loggerKey Key = "context-logger"
)

var baseLogger = logrus.NewEntry(logrus.New())

// GetLogger returns the logger from the context
func GetLogger(ctx Context) logrus.FieldLogger {
	return ctx.Value(loggerKey).(logrus.FieldLogger)
}

// SetLogLevel sets the log level for the logger
func SetLogLevel(level logrus.Level) {
	baseLogger.Logger.SetLevel(level)
}
