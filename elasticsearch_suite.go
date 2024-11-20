package testkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/sirupsen/logrus"

	"github.com/bdpiprava/testkit/context"
	"github.com/bdpiprava/testkit/internal"
)

const createIndexBodyTemplate = `{
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
}`

// RequireElasticSearchClient returns the elasticsearch client
func (s *Suite) RequireElasticSearchClient() *elasticsearch.Client {
	if esClient == nil {
		s.T().Fatalf("elasticsearch client is not initialized")
	}
	return esClient
}

// ElasticSearchCreateIndex creates a new index
func (s *Suite) ElasticSearchCreateIndex(
	index string,
	numberOfShards,
	numberOfReplicas int,
	dynamic bool,
	properties map[string]any,
) error {
	s.esIndices[s.T().Name()] = append(s.esIndices[s.T().Name()], index)

	propBytes, err := json.Marshal(properties)
	s.Require().NoError(err)
	bb := []byte(fmt.Sprintf(createIndexBodyTemplate, numberOfShards, numberOfReplicas, dynamic, string(propBytes)))

	req := esapi.IndicesCreateRequest{
		Index: index,
		Body:  bytes.NewReader(bb),
	}

	ctx := s.GetContext()
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

// ElasticSearchIndexExists checks if the index exists
func (s *Suite) ElasticSearchIndexExists(name string) bool {
	indices := s.ElasticSearchFindIndices(name)
	for _, index := range indices {
		if strings.ToLower(index.Name) == strings.ToLower(name) {
			return true
		}
	}
	return false
}

// ElasticSearchCloseIndices closes the esIndices
func (s *Suite) ElasticSearchCloseIndices(indices ...string) {
	ctx := s.GetContext()
	_, err := esClient.Indices.Close(indices, esClient.Indices.Close.WithContext(ctx))
	s.Require().NoError(err)
}

// ElasticSearchFindIndices returns matching esIndices sorted by name
func (s *Suite) ElasticSearchFindIndices(pattern string) internal.Indices {
	ctx := s.GetContext()
	resp, err := esClient.Cat.Indices(
		esClient.Cat.Indices.WithContext(ctx),
		esClient.Cat.Indices.WithIndex(pattern),
		esClient.Cat.Indices.WithFormat("json"),
	)
	s.Require().NoError(err)
	defer closeSilently(resp.Body)

	result, err := parseElasticSearchResponse[internal.Indices](resp.StatusCode, resp.Body)
	s.Require().NoError(err)

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })

	return result
}

// ElasticSearchGetIndexSettings returns the settings for the given index
func (s *Suite) ElasticSearchGetIndexSettings(index string) internal.IndexSetting {
	resp, err := esClient.Indices.GetSettings(esClient.Indices.GetSettings.WithIndex(index))
	s.Require().NoError(err)

	if resp.IsError() {
		s.T().Fatalf("failed to get index settings: %v", resp)
	}

	all, _ := io.ReadAll(resp.Body)
	var data internal.GetSettingsResponse
	err = json.Unmarshal(all, &data)
	s.Require().NoError(err)
	return data[index].Settings.Index
}

// ElasticSearchDeleteIndices deletes all esIndices matching the pattern
func (s *Suite) ElasticSearchDeleteIndices(pattern string) {
	indices := s.ElasticSearchFindIndices(pattern)
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

// elasticSearchCleanData cleans the data from the elasticsearch
func (s *Suite) elasticSearchCleanData() {
	if esClient == nil {
		return
	}

	_, _ = esClient.Indices.Delete(s.esIndices[s.T().Name()])
}

// ElasticSearchEventuallyBlockStatus waits until the block status is the expected one
func (s *Suite) ElasticSearchEventuallyBlockStatus(
	indexName string,
	status string,
	timeout,
	interval time.Duration,
) {
	s.Eventually(s.checkBlockStatusFn(indexName, status), timeout, interval)
}

// ElasticSearchDeleteByQuery deletes documents matching the provided query.
func (s *Suite) ElasticSearchDeleteByQuery(query string, indices ...string) {
	ctx := s.GetContext()
	log := context.GetLogger(*ctx).WithFields(logrus.Fields{
		"query":   query,
		"indices": indices,
	})

	log.Info("deleting by query")
	resp, err := esClient.DeleteByQuery(
		indices,
		strings.NewReader(query),
		esClient.DeleteByQuery.WithContext(ctx),
	)
	s.Require().NoError(err)
	defer closeSilently(resp.Body)

	result, err := parseElasticSearchResponse[internal.QueryResponse](resp.StatusCode, resp.Body)
	s.Require().NoError(err, "failed to delete by query")

	log.Infof("deleted %d documents", len(result.Hits.Hits))
}

// ElasticSearchSearchByQuery searches for documents matching the provided query.
func (s *Suite) ElasticSearchSearchByQuery(query string, index string) internal.QueryResponse {
	ctx := s.GetContext()
	log := context.GetLogger(*ctx).WithFields(logrus.Fields{
		"query": query,
		"index": index,
	})

	log.Info("deleting by query")
	resp, err := esClient.Search(
		esClient.Search.WithIndex(index),
		esClient.Search.WithBody(strings.NewReader(query)),
	)
	s.Require().NoError(err)
	defer closeSilently(resp.Body)

	result, err := parseElasticSearchResponse[internal.QueryResponse](resp.StatusCode, resp.Body)
	s.Require().NoError(err, "failed to search by query")

	log.Infof("found %d documents", len(result.Hits.Hits))
	return result
}

// ElasticSearchSearchCreateDocument creates a new document in the provided index
func (s *Suite) ElasticSearchSearchCreateDocument(index string, document map[string]any) {
	ctx := s.GetContext()
	log := context.GetLogger(*ctx).WithFields(logrus.Fields{
		"index": index,
	})

	log.Info("creating document")
	content, err := json.Marshal(document)
	s.Require().NoError(err)

	resp, err := esClient.Index(
		index,
		bytes.NewReader(content),
		esClient.Index.WithContext(ctx),
	)
	s.Require().NoError(err)
	defer closeSilently(resp.Body)

	if resp.IsError() {
		s.T().Fatalf("failed to create document: %v", resp)
	}

	log.Info("document created")
}

func (s *Suite) checkBlockStatusFn(indexName string, status string) func() bool {
	return func() bool {
		return s.ElasticSearchGetIndexSettings(indexName).Blocks.Write == status
	}
}

func closeSilently(closable io.Closer) {
	if closable != nil {
		_ = closable.Close()
	}
}

func parseElasticSearchResponse[T any](statusCode int, body io.ReadCloser) (T, error) {
	var result T
	if statusCode == http.StatusNotFound {
		return result, nil
	}

	// If the status code is not 2xx, return an error
	if statusCode > 299 {
		return result, fmt.Errorf("received status code: %d", statusCode)
	}

	err := json.NewDecoder(body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, err
}
