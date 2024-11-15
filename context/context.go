package context

import (
	"context"

	"github.com/bdpiprava/testkit/internal"
	"github.com/sirupsen/logrus"
)

// Key is a type for context key
type Key string

// Context for testkit
type Context struct {
	context.Context
}

// configRoot is the root of logger config
type configRoot struct {
	LogLevel string `yaml:"log_level"` // LogLevel is the log level
}

var baseLogger = logrus.NewEntry(logrus.New())

func init() {
	config, err := internal.ReadConfigAs[configRoot]()
	if err != nil {
		baseLogger.WithError(err).Warn("failed to read log config, initializing with default")
	}

	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		baseLogger.WithError(err).Warn("failed to parse log level, initializing with default")
		level = logrus.InfoLevel
	}

	baseLogger.Logger.SetLevel(level)
}

// NewContext returns a new context with a logger
func NewContext(name string) *Context {
	logger := baseLogger.WithField("name", name)
	ctx := &Context{
		Context: context.WithValue(context.Background(), loggerKey, logger),
	}
	return ctx
}

// SetData sets the data in the context
func (s *Context) SetData(key Key, value any) {
	s.Context = context.WithValue(s.Context, key, value)
}
