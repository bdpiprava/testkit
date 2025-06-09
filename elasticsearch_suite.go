package testkit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esapi"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/bdpiprava/testkit/search"
)

const successStatusCode = 299

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
	CreateDocument(index, docID string, document map[string]any) error
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
	log := s.log.WithFields(logrus.Fields{
		"index":    index,
		"settings": settings,
	})

	log.Debug("creating index")
	bb, err := settings.GetBody()
	if err != nil {
		log.Debug("failed create settings body")
		return err
	}
	req := esapi.IndicesCreateRequest{
		Index: index,
		Body:  bytes.NewReader([]byte(bb)),
	}

	log.Debug("executing create index request")
	ctx := context.Background()
	resp, err := req.Do(ctx, s.client)
	if err != nil {
		log.Debug("failed to execute create index request")
		return err
	}
	defer closeSilently(resp.Body)

	if resp.IsError() {
		log.Debugf("failed to create index: %v", resp.String())
		return fmt.Errorf("failed to create index: %s", resp.String())
	}

	return nil
}

// CloseIndices closes the indices
func (s *elasticSearch) CloseIndices(indices ...string) {
	log := s.log.WithFields(logrus.Fields{
		"indices": indices,
	})
	log.Debug("closing indices")
	resp, err := s.client.Indices.Close(indices)
	if err != nil {
		log.Debug("failed to close indices")
		return
	}
	defer closeSilently(resp.Body)
	if resp.IsError() {
		log.Debugf("failed to close indices: %v", resp.String())
		return
	}
}

// IndexExists checks if the index exists
func (s *elasticSearch) IndexExists(name string) (bool, error) {
	log := s.log.WithFields(logrus.Fields{
		"index": name,
	})

	log.Debug("checking if index exists")
	indices, err := s.FindIndices(name)
	if err != nil {
		log.Debug("failed to find indices")
		return false, err
	}

	for _, index := range indices {
		if strings.EqualFold(index.Name, name) {
			return true, nil
		}
	}
	return false, nil
}

// FindIndices returns matching esIndices sorted by name
func (s *elasticSearch) FindIndices(pattern string) (search.Indices, error) {
	log := s.log.WithFields(logrus.Fields{
		"index": pattern,
	})

	log.Debug("finding indices")
	resp, err := s.client.Cat.Indices(
		s.client.Cat.Indices.WithContext(context.Background()),
		s.client.Cat.Indices.WithIndex(pattern),
		s.client.Cat.Indices.WithFormat("json"),
	)
	if err != nil {
		log.Debug("failed to find indices")
		return nil, err
	}
	defer closeSilently(resp.Body)

	result, err := parseElasticSearchResponse[search.Indices](resp.StatusCode, resp.Body)
	if err != nil {
		log.Debug("failed to parse indices")
		return nil, err
	}

	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// GetIndexSettings returns the settings for the given index
func (s *elasticSearch) GetIndexSettings(index string) (search.IndexSetting, error) {
	log := s.log.WithFields(logrus.Fields{
		"index": index,
	})

	log.Debug("getting index settings")
	resp, err := s.client.Indices.GetSettings(
		s.client.Indices.GetSettings.WithIndex(index),
	)
	if err != nil {
		log.Debug("failed to get index settings")
		return search.IndexSetting{}, err
	}
	defer closeSilently(resp.Body)

	result, err := parseElasticSearchResponse[search.GetSettingsResponse](resp.StatusCode, resp.Body)
	if err != nil {
		log.Debug("failed to parse index settings")
		return search.IndexSetting{}, err
	}
	return result[index].Settings.Index, nil
}

// DeleteIndices deletes all esIndices matching the pattern
func (s *elasticSearch) DeleteIndices(pattern string) error {
	log := s.log.WithFields(logrus.Fields{
		"index": pattern,
	})

	indices, err := s.FindIndices(pattern)
	if err != nil {
		return err
	}

	if len(indices) == 0 {
		s.log.Debug("no indices to delete, returning")
		return nil
	}

	log.Debugf("indices found for deletion: %v", len(indices))
	list := make([]string, 0, len(indices))
	for _, index := range indices {
		list = append(list, index.Name)
	}

	log.Debugf("deleting indices: %v", list)
	_, err = s.client.Indices.Delete(list)
	return err
}

// DeleteByQuery deletes documents matching the provided query.
func (s *elasticSearch) DeleteByQuery(indices []string, query string) error {
	log := s.log.WithFields(logrus.Fields{
		"indices": indices,
		query:     query,
	})

	log.Debug("deleting by query")
	resp, err := s.client.DeleteByQuery(indices, strings.NewReader(query))
	if err != nil {
		log.Debug("failed to delete by query")
		return errors.Wrapf(err, "failed to delete by query")
	}
	defer closeSilently(resp.Body)

	result, err := parseElasticSearchResponse[search.QueryResponse](resp.StatusCode, resp.Body)
	if err != nil {
		log.Debug("failed to parse delete by query response")
		return errors.Wrapf(err, "failed to delete by query: %s", resp.String())
	}

	log.Infof("deleted document count: %d", len(result.Hits.Hits))
	return nil
}

// SearchByQuery searches for documents matching the provided query.
func (s *elasticSearch) SearchByQuery(index string, query string) (search.QueryResponse, error) {
	log := s.log.WithFields(logrus.Fields{
		"index": index,
		"query": query,
	})

	log.Debug("searching by query")
	var result search.QueryResponse
	resp, err := s.client.Search(
		s.client.Search.WithIndex(index),
		s.client.Search.WithBody(strings.NewReader(query)),
	)
	if err != nil {
		log.Debug("failed to search by query")
		return result, errors.Wrapf(err, "failed to search by query")
	}
	defer closeSilently(resp.Body)

	result, err = parseElasticSearchResponse[search.QueryResponse](resp.StatusCode, resp.Body)
	if err != nil {
		log.Debug("failed to parse search by query response")
		return result, errors.Wrapf(err, "failed to search by query: %s", resp.String())
	}

	log.Infof("found %d documents", len(result.Hits.Hits))
	return result, nil
}

// CreateDocument creates a new document in the provided index
func (s *elasticSearch) CreateDocument(index, docID string, document map[string]any) error {
	log := s.log.WithFields(logrus.Fields{
		"index": index,
	})

	log.Debug("marshalling document content")
	content, err := json.Marshal(document)
	if err != nil {
		log.Debug("failed to marshal document content")
		return err
	}

	log.Debug("creating document")
	resp, err := s.client.Index(
		index,
		bytes.NewReader(content),
		s.client.Index.WithRefresh("true"),
		s.client.Index.WithDocumentID(docID),
	)
	if err != nil {
		log.Debug("failed to create document")
		return errors.Wrapf(err, "failed to create document")
	}
	defer closeSilently(resp.Body)

	if resp.IsError() {
		log.Debugf("failed to create document: %v", resp.String())
		return errors.Errorf("failed to create document: %v", resp.String())
	}
	log.Debugf("document created")

	return nil
}

func closeSilently(closable io.Closer) {
	if closable == nil || (reflect.ValueOf(closable).Kind() == reflect.Ptr && reflect.ValueOf(closable).IsNil()) {
		return
	}

	_ = closable.Close()
}

func parseElasticSearchResponse[T any](statusCode int, body io.ReadCloser) (T, error) {
	var result T
	if statusCode == http.StatusNotFound {
		return result, nil
	}

	// If the status code is not 2xx, return an error
	if statusCode > successStatusCode {
		return result, fmt.Errorf("received status code: %d", statusCode)
	}

	err := json.NewDecoder(body).Decode(&result)
	if err != nil {
		return result, err
	}

	return result, err
}
