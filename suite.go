package testkit

import (
	"context"
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
)

const defaultWiremockAddress = "http://localhost:8080"

var esClient *elasticsearch.Client
var osClient *opensearch.Client
var wiremockClient *wiremock.Client
var suiteConfig *internal.SuiteConfig

type Suite struct {
	*assert.Assertions

	mu  sync.RWMutex
	t   *testing.T
	s   TestingSuite
	ctx context.Context
	r   *require.Assertions
	l   logrus.FieldLogger
	i   bool

	// Used by postgres_suite.go
	postgresDB *internal.PostgresDB
	testDBs    map[string][]string

	// Used by elasticsearch_suite.go
	esIndices map[string][]string

	// Used by opensearch_suite.go
	osIndices map[string][]string

	// Used by kafka_suite.go
	kafkaServers   map[string]*kafka.MockCluster
	kafkaConsumers []*kafka.Consumer
}

// TearDownSuite perform the cleanup of the database
func (s *Suite) TearDownSuite() {
	defer s.cleanDatabase()
	defer s.cleanKafkaResources()
	defer s.elasticSearchCleanData()
	defer s.openSearchCleanData()
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
	s.t = t
	s.mu.Unlock()

	// suite is not initialised yet
	if !s.i {
		return
	}

	if _, ok := s.testDBs[s.T().Name()]; !ok {
		s.testDBs[s.T().Name()] = make([]string, 0)
	}

	if _, ok := s.esIndices[s.T().Name()]; !ok {
		s.esIndices[s.T().Name()] = make([]string, 0)
	}

	if _, ok := s.osIndices[s.T().Name()]; !ok {
		s.osIndices[s.T().Name()] = make([]string, 0)
	}
}

// SetS needs to set the current test suite as parent to get access to the parent methods
func (s *Suite) SetS(suite TestingSuite) {
	s.s = suite
	err := s.initializeSuite(s.T())
	if err != nil {
		s.T().Fatalf("failed to initialize suite: %s", err)
	}
	s.i = true
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

// initializeSuite initialize the suite
func (s *Suite) initializeSuite(t *testing.T) error {
	s.ctx = context.Background()
	s.kafkaServers = make(map[string]*kafka.MockCluster)
	s.kafkaConsumers = make([]*kafka.Consumer, 0)
	s.esIndices = make(map[string][]string)
	s.osIndices = make(map[string][]string)
	s.testDBs = make(map[string][]string)

	s.Assertions = assert.New(t)
	s.r = require.New(t)

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

	if initialiseElasticSearch(config.ElasticSearch, s.l) != nil {
		return err
	}

	if initialiseOpenSearch(config.OpenSearch, s.l) != nil {
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
