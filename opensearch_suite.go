package testkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/bdpiprava/testkit/search"
)

// openSearch is a wrapper around the elasticsearch client
type openSearch struct {
	client *opensearch.Client
	log    logrus.FieldLogger
}

// RequireOpenSearch returns the opensearch client
func (s *Suite) RequireOpenSearch() SearchClient {
	if osClient == nil {
		s.T().Fatalf("opensearch client is not initialized")
	}
	return &openSearch{
		client: osClient,
		log:    s.l,
	}
}

// CreateIndex creates a new index
func (s *openSearch) CreateIndex(index string, settings search.CreateIndexSettings) error {
	bb, err := settings.GetBody()
	if err != nil {
		return err
	}
	req := esapi.IndicesCreateRequest{
		Index: index,
		Body:  bytes.NewReader([]byte(bb)),
	}

	ctx := context.Background()
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

// CloseIndices closes the indices
func (s *openSearch) CloseIndices(indices ...string) {
	_, _ = esClient.Indices.Close(indices)
}

// IndexExists checks if the index exists
func (s *openSearch) IndexExists(name string) (bool, error) {
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
func (s *openSearch) FindIndices(pattern string) (search.Indices, error) {
	resp, err := esClient.Cat.Indices(
		esClient.Cat.Indices.WithContext(context.Background()),
		esClient.Cat.Indices.WithIndex(pattern),
		esClient.Cat.Indices.WithFormat("json"),
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
func (s *openSearch) GetIndexSettings(index string) (search.IndexSetting, error) {
	resp, err := esClient.Indices.GetSettings(
		esClient.Indices.GetSettings.WithIndex(index),
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
func (s *openSearch) DeleteIndices(pattern string) error {
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
	_, err = esClient.Indices.Delete(list)
	return err
}

// DeleteByQuery deletes documents matching the provided query.
func (s *openSearch) DeleteByQuery(indices []string, query string) error {
	resp, err := esClient.DeleteByQuery(indices, strings.NewReader(query))
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
func (s *openSearch) SearchByQuery(index string, query string) (search.QueryResponse, error) {
	var result search.QueryResponse
	resp, err := esClient.Search(
		esClient.Search.WithIndex(index),
		esClient.Search.WithBody(strings.NewReader(query)),
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
func (s *openSearch) CreateDocument(index string, document map[string]any) error {
	log := s.log.WithFields(logrus.Fields{
		"index": index,
	})

	log.Debug("marshalling document content")
	content, err := json.Marshal(document)
	if err != nil {
		return err
	}

	log.Debug("creating document")
	resp, err := esClient.Index(
		index,
		bytes.NewReader(content),
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
