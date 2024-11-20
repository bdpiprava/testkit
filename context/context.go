package context

import (
	"context"
)

// Key is a type for context key
type Key string

// Context for testkit
type Context struct {
	context.Context
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
