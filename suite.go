package testkit

import (
	"context"
	"flag"
	"strings"
	"sync"
	"testing"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wiremock/go-wiremock"

	"github.com/bdpiprava/testkit/internal"
	"github.com/bdpiprava/testkit/suite"
)

var (
	allTestsFilter = func(_, _ string) (bool, error) { return true, nil }
	matchMethod    = flag.String("testkit.m", "", "regular expression to select tests of the testify suite to run")
	esClient       *elasticsearch.Client
	osClient       *opensearch.Client
	wiremockClient *wiremock.Client
	suiteConfig    *internal.SuiteConfig
)

const defaultWiremockAddress = "http://localhost:8080"

type Suite struct {
	suite.Suite
	*assert.Assertions

	mu  sync.RWMutex
	t   *testing.T
	ctx context.Context
	r   *require.Assertions
	l   logrus.FieldLogger

	// Used by postgres_suite.go
	postgresDB *internal.PostgresDB

	// Used by kafka_suite.go
	kafkaServers   map[string]*kafka.MockCluster
	kafkaConsumers []*kafka.Consumer

	// Parent suite to have access to the implemented methods of parent struct
	s TestingSuite
}

// T retrieves the current *testing.T context
func (s *Suite) T() *testing.T {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.t
}

// SetT sets the current *testing.T context
func (s *Suite) SetT(t *testing.T) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.t = t
	s.Assertions = assert.New(t)
	s.r = require.New(t)
}

// DoOnce setup the suite
func (s *Suite) DoOnce(t *testing.T) error {
	return s.initializeSuite(t)
}

// SetS needs to set the current test suite as parent to get access to the parent methods
func (s *Suite) SetS(suite TestingSuite) {
	s.s = suite
}

// TearDownSuite perform the cleanup of the database
func (s *Suite) TearDownSuite() {
	defer s.cleanDatabase()
	defer s.cleanKafkaResources()
}

// GetContext returns the context created for the current test, if not exists then creates a new context and returns
func (s *Suite) GetContext() context.Context {
	return s.ctx
}

// Require returns a require.Assertions object to make assertions
func (s *Suite) Require() *require.Assertions {
	return s.r
}

// Logger returns the logger
func (s *Suite) Logger() logrus.FieldLogger {
	return s.l
}

// Run provides suite functionality around golang subtests.  It should be
// called in place of t.Run(name, func(t *testing.T)) in test suite code.
// The passed-in func will be executed as a subtest with a fresh instance of t.
// Provides compatibility with go test pkg -run TestSuite/TestName/SubTestName.
func (s *Suite) Run(name string, subtest func()) bool {
	oldT := s.T()

	return oldT.Run(name, func(t *testing.T) {
		s.SetT(t)
		defer s.SetT(oldT)

		defer recoverAndFailOnPanic(t)

		if setupSubTest, ok := s.s.(SetupTest); ok {
			setupSubTest.SetupTest()
		}

		if tearDownSubTest, ok := s.s.(TearDownTest); ok {
			defer tearDownSubTest.TearDownTest()
		}

		subtest()
	})
}

// initializeSuite initialize the suite
func (s *Suite) initializeSuite(_ *testing.T) error {
	s.ctx = context.Background()
	s.kafkaServers = make(map[string]*kafka.MockCluster)
	s.kafkaConsumers = make([]*kafka.Consumer, 0)

	logger := logrus.New()
	config, err := getConfig()
	if err != nil {
		return err
	}
	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logger.WithError(err).Warn("failed to parse log level, initializing with default")
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)
	s.l = logrus.NewEntry(logger)

	_, err = internal.InitialiseDatabase(*config, s.l)
	if err != nil && !errors.Is(err, internal.ErrMissingGoMigrateConfig) {
		return err
	}

	if wiremockClient == nil {
		wiremockClient = wiremock.NewClient(config.APIMockConfig.Address)
	}

	if err = initialiseElasticSearch(config.ElasticSearch, s.l); err != nil {
		return err
	}

	if err = initialiseOpenSearch(config.OpenSearch, s.l); err != nil {
		return err
	}

	return nil
}

// initialiseElasticSearch initializes the elastic search client
func initialiseElasticSearch(config *internal.ElasticSearchConfig, log logrus.FieldLogger) (err error) {
	if esClient != nil {
		return nil
	}

	if config == nil {
		return nil
	}

	esClient, err = elasticsearch.NewClient(elasticsearch.Config{
		Addresses: strings.Split(config.Addresses, ","),
		Username:  config.Username,
		Password:  config.Password,
	})

	if err != nil {
		return err
	}

	res, err := esClient.Ping()
	if err != nil {
		return err
	}

	log.Infof("Connected to elastic search cluster: %v", res.String())
	return nil
}

// initialiseOpenSearch initializes the open search client
func initialiseOpenSearch(config *internal.ElasticSearchConfig, log logrus.FieldLogger) (err error) {
	if osClient != nil {
		return nil
	}

	if config == nil {
		return nil
	}

	osClient, err = opensearch.NewClient(opensearch.Config{
		Addresses: strings.Split(config.Addresses, ","),
		Username:  config.Username,
		Password:  config.Password,
	})

	if err != nil {
		return err
	}

	res, err := osClient.Ping()
	if err != nil {
		return err
	}

	log.Infof("Connected to open search cluster: %v", res.String())
	return nil
}

// getConfig reads the suite configuration from the file
func getConfig() (*internal.SuiteConfig, error) {
	if suiteConfig != nil {
		return suiteConfig, nil
	}

	cfg, err := internal.ReadConfigAs[internal.SuiteConfig]()
	if err != nil {
		return nil, err
	}

	if cfg.APIMockConfig == nil {
		cfg.APIMockConfig = &internal.APIMockConfig{Address: defaultWiremockAddress}
	}

	suiteConfig = &cfg
	return suiteConfig, nil
}
