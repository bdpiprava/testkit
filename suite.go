package testkit

import (
	"strings"
	"sync"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/elastic/go-elasticsearch/v7"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/wiremock/go-wiremock"

	"github.com/bdpiprava/testkit/context"
	"github.com/bdpiprava/testkit/internal"
)

var esClient *elasticsearch.Client
var wiremockAddress = "http://localhost:8080"
var wiremockClient *wiremock.Client
var suiteConfig internal.SuiteConfig

type Suite struct {
	context.CtxSuite
	initializer sync.Once

	// Used by postgres_suite.go
	postgresDB *internal.PostgresDB

	// Used by elasticsearch_suite.go
	indices map[string][]string

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

		if s.indices == nil {
			s.indices = make(map[string][]string)
		}

		if _, ok := s.indices[s.T().Name()]; !ok {
			s.indices[s.T().Name()] = make([]string, 0)
		}

		ctx := s.GetContext(s.T().Name())
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
				s.FailNowf("Failed to connect to elastic search, received error response %s", res.String())
			}
			log.Infof("Connection established %s", res.String())
		}
	})
}

// TearDownSuite perform the cleanup of the database
func (s *Suite) TearDownSuite() {
	defer s.cleanDatabase()
	defer s.cleanKafkaResources()
	defer s.cleanElasticSearchData()
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
