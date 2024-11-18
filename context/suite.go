package context

import (
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

// ContextSuite for context mapping
//
//goland:noinspection GoNameStartsWithPackageName
type ContextSuite struct {
	suite.Suite
	ctxMap      map[string]*Context
	initializer sync.Once
}

// Initialize the context for the suite
func (s *ContextSuite) Initialize(name string) (err error) {
	log := baseLogger.WithFields(logrus.Fields{
		"func": "Initialize",
		"name": name,
	})
	log.Trace("Start")

	s.initializer.Do(func() {
		log.Trace("[Do] Initializing")
		s.ctxMap = make(map[string]*Context)
	})
	log.Trace("[Do] Initialized")

	if _, ok := s.ctxMap[name]; !ok {
		log.Trace("Creating new context")
		s.ctxMap[name] = NewContext(name)
	}

	log.Trace("Done")
	return err
}

// GetContext returns the context created for the current test, if not exists then creates a new context and returns
func (s *ContextSuite) GetContext(name string) *Context {
	log := baseLogger.WithFields(logrus.Fields{
		"func": "GetContext",
		"name": name,
	})

	log.Trace("Start")
	if ctx, ok := s.ctxMap[name]; ok {
		log.Trace("Found context, returning")
		return ctx
	}
	log.Trace("Creating new context")
	ctx := NewContext(name)
	s.ctxMap[name] = ctx

	log.Trace("Done, returning new context")
	return ctx
}
