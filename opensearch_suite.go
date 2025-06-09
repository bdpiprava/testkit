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
func (s *openSearch) CloseIndices(indices ...string) {
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
func (s *openSearch) IndexExists(name string) (bool, error) {
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
func (s *openSearch) FindIndices(pattern string) (search.Indices, error) {
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
func (s *openSearch) GetIndexSettings(index string) (search.IndexSetting, error) {
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
func (s *openSearch) DeleteIndices(pattern string) error {
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
func (s *openSearch) DeleteByQuery(indices []string, query string) error {
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
func (s *openSearch) SearchByQuery(index string, query string) (search.QueryResponse, error) {
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
func (s *openSearch) CreateDocument(index, docID string, document map[string]any) error {
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
