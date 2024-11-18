package testkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/sirupsen/logrus"

	"github.com/bdpiprava/testkit/context"
	"github.com/bdpiprava/testkit/internal"
)

// elasticConfigRoot is the root configuration for the elastic search client
type elasticConfigRoot struct {
	ElasticSearch *ElasticSearchConfig `json:"elasticsearch"`
}

// ElasticSearchConfig is the configuration for the elastic search client
type ElasticSearchConfig struct {
	Addresses string `json:"addresses"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

var esClient *elasticsearch.Client

func init() {
	config, err := internal.ReadConfigAs[elasticConfigRoot]()
	if err != nil {
		panic(err)
	}

	if config.ElasticSearch == nil {
		return
	}

	esCfg := elasticsearch.Config{
		Addresses: strings.Split(config.ElasticSearch.Addresses, ","),
		Username:  config.ElasticSearch.Username,
		Password:  config.ElasticSearch.Password,
	}

	esClient, err = elasticsearch.NewClient(esCfg)
	if err != nil {
		panic(err)
	}
}

// ElasticSearchSuite is the base test suite for all tests that require an ElasticSearch instance
type ElasticSearchSuite struct {
	context.ContextSuite
	initializer sync.Once
	indices     map[string][]string
}

// CreateIndex creates a new index
func (s *ElasticSearchSuite) CreateIndex(index string, numberOfShards, numberOfReplicas int, dynamic bool, properties map[string]any) error {
	s.initialiseSuite()
	s.indices[s.T().Name()] = append(s.indices[s.T().Name()], index)

	propBytes, err := json.Marshal(properties)
	s.Require().NoError(err)

	bb := []byte(fmt.Sprintf(`{
		"settings": {
			"index": {
				"number_of_shards": %d,
				"number_of_replicas": %d
			}
		},
		"mappings": {
			"dynamic": %t,
			"properties": %s
		}
	}`, numberOfShards, numberOfReplicas, dynamic, string(propBytes)))

	req := esapi.IndicesCreateRequest{
		Index: index,
		Body:  bytes.NewReader(bb),
	}

	ctx := s.GetContext(s.T().Name())
	resp, err := req.Do(ctx, esClient)
	if err != nil {
		return err
	}
	defer closeSilently(resp.Body)

	if resp.IsError() {
		return fmt.Errorf("failed to create index: %s", resp.String())
	}

	return nil
}

// IndexExists checks if the index exists
func (s *ElasticSearchSuite) IndexExists(name string) bool {
	s.initialiseSuite()
	indices := s.FindIndices(name)
	for _, index := range indices {
		if strings.ToLower(index.Name) == strings.ToLower(name) {
			return true
		}
	}
	return false
}

// CloseIndices closes the indices
func (s *ElasticSearchSuite) CloseIndices(indices ...string) {
	s.initialiseSuite()
	ctx := s.GetContext(s.T().Name())
	_, err := esClient.Indices.Close(indices, esClient.Indices.Close.WithContext(ctx))
	s.Require().NoError(err)
}

// FindIndices returns matching indices sorted by name
func (s *ElasticSearchSuite) FindIndices(pattern string) Indices {
	s.initialiseSuite()
	ctx := s.GetContext(s.T().Name())
	resp, err := esClient.Cat.Indices(
		esClient.Cat.Indices.WithContext(ctx),
		esClient.Cat.Indices.WithIndex(pattern),
		esClient.Cat.Indices.WithFormat("json"),
	)
	s.Require().NoError(err)
	defer closeSilently(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return make(Indices, 0)
	}

	if resp.IsError() {
		s.T().Fatalf("failed to get all indices: %v", resp)
	}

	respBytes, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	var result Indices
	err = json.Unmarshal(respBytes, &result)
	s.Require().NoError(err)

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// GetIndexSettings returns the settings for the given index
func (s *ElasticSearchSuite) GetIndexSettings(index string) IndexSetting {
	s.initialiseSuite()
	resp, err := esClient.Indices.GetSettings(esClient.Indices.GetSettings.WithIndex(index))
	s.Require().NoError(err)

	if resp.IsError() {
		s.T().Fatalf("failed to get index settings: %v", resp)
	}

	all, _ := io.ReadAll(resp.Body)
	var data getSettingsResponse
	err = json.Unmarshal(all, &data)
	return data[index].Settings.Index
}

// DeleteIndices deletes all indices matching the pattern
func (s *ElasticSearchSuite) DeleteIndices(pattern string) {
	s.initialiseSuite()
	indices := s.FindIndices(pattern)
	if len(indices) == 0 {
		return
	}
	list := make([]string, 0, len(indices))
	for _, index := range indices {
		list = append(list, index.Name)
	}
	_, err := esClient.Indices.Delete(list)
	s.Require().NoError(err)
}

// TearDownTest deletes the indices created during the test
func (s *ElasticSearchSuite) TearDownTest() {
	_, err := esClient.Indices.Delete(s.indices[s.T().Name()])
	s.Require().NoError(err)
}

// EventuallyBlockStatus waits until the block status is the expected one
func (s *ElasticSearchSuite) EventuallyBlockStatus(indexName string, status string, eventuallyTimeout, eventuallyInterval time.Duration) {
	s.Eventually(s.checkBlockStatusFn(indexName, status), eventuallyTimeout, eventuallyInterval)
}

func (s *ElasticSearchSuite) checkBlockStatusFn(indexName string, status string) func() bool {
	return func() bool {
		return s.GetIndexSettings(indexName).Blocks.Write == status
	}
}

// initialiseSuite initialise the suite
func (s *ElasticSearchSuite) initialiseSuite() {
	s.Require().NoError(s.Initialize(s.T().Name()))
	s.initializer.Do(func() {
		if s.indices == nil {
			s.indices = make(map[string][]string)
		}

		if _, ok := s.indices[s.T().Name()]; !ok {
			s.indices[s.T().Name()] = make([]string, 0)
		}
	})

	ctx := s.GetContext(s.T().Name())
	log := context.GetLogger(*ctx).WithFields(logrus.Fields{
		"test": s.T().Name(),
		"func": "initialiseSuite",
	})

	res, err := esClient.Ping()
	s.Require().NoError(err)
	if res.IsError() {
		log.Errorf("Error: %s", res.String())
		s.FailNowf("Failed to connect to elastic search, received error response %s", res.String())
	}

	log.Infof("Connection established %s", res.String())
}

func closeSilently(closable io.Closer) {
	_ = closable.Close()
}
