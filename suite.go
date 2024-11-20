package testkit

import (
	"strings"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/wiremock/go-wiremock"

	"github.com/bdpiprava/testkit/context"
	"github.com/bdpiprava/testkit/internal"
)

var esClient *elasticsearch.Client
var osClient *opensearch.Client
var wiremockAddress = "http://localhost:8080"
var wiremockClient *wiremock.Client
var suiteConfig internal.SuiteConfig

type Suite struct {
	context.CtxSuite
	initializer sync.Once

	// Used by postgres_suite.go
	postgresDB *internal.PostgresDB

	// Used by elasticsearch_suite.go
	esIndices map[string][]string

	// Used by opensearch_suite.go
	osIndices map[string][]string

	// Used by kafka_suite.go
	kafkaServers   map[string]*kafka.MockCluster
	kafkaConsumers []*kafka.Consumer
}

// SetupSuite initializes the suite
func (s *Suite) SetupSuite() {
	err := s.Initialize(s.T().Name())
	s.Require().NoError(err)

	s.initializer.Do(func() {
		s.kafkaServers = make(map[string]*kafka.MockCluster)
		s.kafkaConsumers = make([]*kafka.Consumer, 0)

		if s.esIndices == nil {
			s.esIndices = make(map[string][]string)
		}

		if s.osIndices == nil {
			s.osIndices = make(map[string][]string)
		}

		if _, ok := s.esIndices[s.T().Name()]; !ok {
			s.esIndices[s.T().Name()] = make([]string, 0)
		}

		if _, ok := s.osIndices[s.T().Name()]; !ok {
			s.osIndices[s.T().Name()] = make([]string, 0)
		}

		ctx := s.GetContext()
		log := context.GetLogger(*ctx).WithFields(logrus.Fields{
			"test": s.T().Name(),
			"func": "initialiseSuite",
		})

		wiremockClient = wiremock.NewClient(wiremockAddress)

		if esClient != nil {
			res, err := esClient.Ping()
			s.Require().NoError(err)
			if res.IsError() {
				log.Errorf("Error: %s", res.String())
				s.T().Fatalf("[ElasticSearch] Failed to connect, received error response %s", res.String())
			}
			log.Infof("[ElasticSearch] Connection established %s", res.String())
		}

		if osClient != nil {
			res, err := osClient.Ping()
			s.Require().NoError(err)
			if res.IsError() {
				log.Errorf("Error: %s", res.String())
				s.T().Fatalf("[OpenSearch] Failed to connect, received error response %s", res.String())
			}
			log.Infof("[OpenSearch] Connection established %s", res.String())
		}
	})
}

// TearDownSuite perform the cleanup of the database
func (s *Suite) TearDownSuite() {
	defer s.cleanDatabase()
	defer s.cleanKafkaResources()
	defer s.elasticSearchCleanData()
	defer s.openSearchCleanData()
}

func init() {
	config, err := internal.ReadConfigAs[internal.SuiteConfig]()
	if err != nil {
		panic(err)
	}

	suiteConfig = config
	_, err = internal.InitialiseDatabase(config)
	if err != nil && !errors.Is(err, internal.ErrMissingGoMigrateConfig) {
		panic(err)
	}

	if config.ElasticSearch != nil {
		esClient, err = elasticsearch.NewClient(elasticsearch.Config{
			Addresses: strings.Split(config.ElasticSearch.Addresses, ","),
			Username:  config.ElasticSearch.Username,
			Password:  config.ElasticSearch.Password,
		})

		if err != nil {
			panic(err)
		}
	}

	if config.OpenSearch != nil {
		osClient, err = opensearch.NewClient(opensearch.Config{
			Addresses: strings.Split(config.OpenSearch.Addresses, ","),
			Username:  config.OpenSearch.Username,
			Password:  config.OpenSearch.Password,
		})

		if err != nil {
			panic(err)
		}
	}

	if config.APIMockConfig != nil {
		wiremockAddress = config.APIMockConfig.Address
	}

	level, err := logrus.ParseLevel(config.LogLevel)
	if err != nil {
		logrus.New().WithError(err).Warn("failed to parse log level, initializing with default")
		level = logrus.InfoLevel
	}

	context.SetLogLevel(level)
}
