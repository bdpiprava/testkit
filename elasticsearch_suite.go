package testkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/bdpiprava/testkit/search"
)

// SearchClient is the interface for the search client
type SearchClient interface {
	// CreateIndex creates a new index
	CreateIndex(index string, settings search.CreateIndexSettings) error
	// CloseIndices closes the indices
	CloseIndices(indices ...string)
	// IndexExists checks if the index exists
	IndexExists(name string) (bool, error)
	// FindIndices returns matching esIndices sorted by name
	FindIndices(pattern string) (search.Indices, error)
	// GetIndexSettings returns the settings for the given index
	GetIndexSettings(index string) (search.IndexSetting, error)
	// DeleteIndices deletes all esIndices matching the pattern
	DeleteIndices(pattern string) error
	// DeleteByQuery deletes documents matching the provided query.
	DeleteByQuery(indices []string, query string) error
	// SearchByQuery searches for documents matching the provided query.
	SearchByQuery(index string, query string) (search.QueryResponse, error)
	// CreateDocument creates a new document in the provided index
	CreateDocument(index string, document map[string]any) error
}

// ElasticSearch is a wrapper around the elasticsearch client
type elasticSearch struct {
	client *elasticsearch.Client
	log    logrus.FieldLogger
}

// RequireElasticSearch returns the elasticsearch client
func (s *Suite) RequireElasticSearch() SearchClient {
	if esClient == nil {
		s.T().Fatalf("elasticsearch client is not initialized")
	}
	return &elasticSearch{
		client: esClient,
		log:    s.Logger(),
	}
}

// CreateIndex creates a new index
func (s *elasticSearch) CreateIndex(index string, settings search.CreateIndexSettings) error {
	bb, err := settings.GetBody()
	if err != nil {
		return err
	}
	req := esapi.IndicesCreateRequest{
		Index: index,
		Body:  bytes.NewReader([]byte(bb)),
	}

	ctx := context.Background()
	resp, err := req.Do(ctx, s.client)
	if err != nil {
		return err
	}
	defer closeSilently(resp.Body)

	if resp.IsError() {
		return fmt.Errorf("failed to create index: %s", resp.String())
	}

	return nil
}

// CloseIndices closes the indices
func (s *elasticSearch) CloseIndices(indices ...string) {
	_, _ = s.client.Indices.Close(indices)
}

// IndexExists checks if the index exists
func (s *elasticSearch) IndexExists(name string) (bool, error) {
	indices, err := s.FindIndices(name)
	if err != nil {
		return false, err
	}

	for _, index := range indices {
		if strings.ToLower(index.Name) == strings.ToLower(name) {
			return true, nil
		}
	}
	return false, nil
}

// FindIndices returns matching esIndices sorted by name
func (s *elasticSearch) FindIndices(pattern string) (search.Indices, error) {
	resp, err := s.client.Cat.Indices(
		s.client.Cat.Indices.WithContext(context.Background()),
		s.client.Cat.Indices.WithIndex(pattern),
		s.client.Cat.Indices.WithFormat("json"),
	)
	if err != nil {
		return nil, err
	}
	defer closeSilently(resp.Body)

	result, err := parseElasticSearchResponse[search.Indices](resp.StatusCode, resp.Body)
	if err != nil {
		return nil, err
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// GetIndexSettings returns the settings for the given index
func (s *elasticSearch) GetIndexSettings(index string) (search.IndexSetting, error) {
	resp, err := s.client.Indices.GetSettings(
		s.client.Indices.GetSettings.WithIndex(index),
	)
	if err != nil {
		return search.IndexSetting{}, err
	}
	defer closeSilently(resp.Body)

	result, err := parseElasticSearchResponse[search.GetSettingsResponse](resp.StatusCode, resp.Body)
	if err != nil {
		return search.IndexSetting{}, err
	}
	return result[index].Settings.Index, nil
}

// DeleteIndices deletes all esIndices matching the pattern
func (s *elasticSearch) DeleteIndices(pattern string) error {
	indices, err := s.FindIndices(pattern)
	if err != nil {
		return err
	}

	if len(indices) == 0 {
		return nil
	}

	list := make([]string, 0, len(indices))
	for _, index := range indices {
		list = append(list, index.Name)
	}
	_, err = s.client.Indices.Delete(list)
	return err
}

// DeleteByQuery deletes documents matching the provided query.
func (s *elasticSearch) DeleteByQuery(indices []string, query string) error {
	resp, err := s.client.DeleteByQuery(indices, strings.NewReader(query))
	if err != nil {
		return errors.Wrapf(err, "failed to delete by query")
	}
	defer closeSilently(resp.Body)

	result, err := parseElasticSearchResponse[search.QueryResponse](resp.StatusCode, resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to delete by query: %s", resp.String())
	}

	s.log.Infof("deleted document count: %d", len(result.Hits.Hits))
	return nil
}

// SearchByQuery searches for documents matching the provided query.
func (s *elasticSearch) SearchByQuery(index string, query string) (search.QueryResponse, error) {
	var result search.QueryResponse
	resp, err := s.client.Search(
		s.client.Search.WithIndex(index),
		s.client.Search.WithBody(strings.NewReader(query)),
	)
	if err != nil {
		return result, errors.Wrapf(err, "failed to search by query")
	}
	defer closeSilently(resp.Body)

	result, err = parseElasticSearchResponse[search.QueryResponse](resp.StatusCode, resp.Body)
	if err != nil {
		return result, errors.Wrapf(err, "failed to search by query: %s", resp.String())
	}

	s.log.Infof("found %d documents", len(result.Hits.Hits))
	return result, nil
}

// CreateDocument creates a new document in the provided index
func (s *elasticSearch) CreateDocument(index string, document map[string]any) error {
	log := s.log.WithFields(logrus.Fields{
		"index": index,
	})

	log.Debug("marshalling document content")
	content, err := json.Marshal(document)
	if err != nil {
		return err
	}

	options := make([]func(*esapi.IndexRequest), 0)
	options = append(options, s.client.Index.WithRefresh("true"))
	if id, ok := document["document_id"]; ok {
		options = append(options, s.client.Index.WithDocumentID(id.(string)))
	}

	log.Debug("creating document")
	resp, err := s.client.Index(
		index,
		bytes.NewReader(content),
		options...,
	)
	if err != nil {
		return errors.Wrapf(err, "failed to create document")
	}
	defer closeSilently(resp.Body)

	if resp.IsError() {
		return errors.Errorf("failed to create document: %v", resp.String())
	}
	log.Debugf("document created")

	return nil
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
