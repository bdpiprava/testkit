package testkit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/sirupsen/logrus"

	"github.com/bdpiprava/testkit/context"
	"github.com/bdpiprava/testkit/internal"
)

// RequireOpenSearchClient returns the opensearch client
func (s *Suite) RequireOpenSearchClient() *opensearch.Client {
	if osClient == nil {
		s.T().Fatalf("opensearch client is not initialized")
	}
	return osClient
}

// OpenSearchCreateIndex creates a new index in opensearch
func (s *Suite) OpenSearchCreateIndex(
	index string,
	numberOfShards,
	numberOfReplicas int,
	dynamic bool,
	properties map[string]any,
) error {
	s.osIndices[s.T().Name()] = append(s.osIndices[s.T().Name()], index)

	propBytes, err := json.Marshal(properties)
	s.Require().NoError(err)
	bb := []byte(fmt.Sprintf(createIndexBodyTemplate, numberOfShards, numberOfReplicas, dynamic, string(propBytes)))

	resp, err := osClient.Indices.Create(
		index,
		osClient.Indices.Create.WithBody(bytes.NewReader(bb)),
		osClient.Indices.Create.WithContext(s.GetContext()),
	)
	if err != nil {
		return err
	}
	defer closeSilently(resp.Body)

	if resp.IsError() {
		return fmt.Errorf("failed to create index: %s", resp.String())
	}

	return nil
}

// OpenSearchIndexExists checks if the index exists in opensearch
func (s *Suite) OpenSearchIndexExists(name string) bool {
	indices := s.OpenSearchFindIndices(name)
	for _, index := range indices {
		if strings.ToLower(index.Name) == strings.ToLower(name) {
			return true
		}
	}
	return false
}

// OpenSearchCloseIndices closes the opensearch indices
func (s *Suite) OpenSearchCloseIndices(indices ...string) {
	ctx := s.GetContext()
	resp, err := osClient.Indices.Close(indices, osClient.Indices.Close.WithContext(ctx))
	s.Require().NoError(err)
	defer closeSilently(resp.Body)
}

// OpenSearchFindIndices returns matching opensearch indices sorted by name
func (s *Suite) OpenSearchFindIndices(pattern string) internal.Indices {
	ctx := s.GetContext()
	resp, err := osClient.Cat.Indices(
		osClient.Cat.Indices.WithContext(ctx),
		osClient.Cat.Indices.WithIndex(pattern),
		osClient.Cat.Indices.WithFormat("json"),
	)
	s.Require().NoError(err)
	defer closeSilently(resp.Body)

	result, err := parseElasticSearchResponse[internal.Indices](resp.StatusCode, resp.Body)
	s.Require().NoError(err, "failed to find indices")

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// OpenSearchGetIndexSettings returns the settings for the given index
func (s *Suite) OpenSearchGetIndexSettings(index string) internal.IndexSetting {
	resp, err := osClient.Indices.GetSettings(osClient.Indices.GetSettings.WithIndex(index))
	s.Require().NoError(err)
	defer closeSilently(resp.Body)

	if resp.IsError() {
		s.T().Fatalf("failed to get index settings: %v", resp)
	}

	all, _ := io.ReadAll(resp.Body)
	var data internal.GetSettingsResponse
	err = json.Unmarshal(all, &data)
	s.Require().NoError(err)
	return data[index].Settings.Index
}

// OpenSearchDeleteByQuery deletes documents matching the provided query.
func (s *Suite) OpenSearchDeleteByQuery(query string, indices ...string) {
	ctx := s.GetContext()
	log := context.GetLogger(*ctx).WithFields(logrus.Fields{
		"query":   query,
		"indices": indices,
	})

	log.Info("deleting by query")
	resp, err := osClient.DeleteByQuery(
		indices,
		strings.NewReader(query),
		osClient.DeleteByQuery.WithContext(ctx),
	)
	s.Require().NoError(err)
	defer closeSilently(resp.Body)

	result, err := parseElasticSearchResponse[internal.QueryResponse](resp.StatusCode, resp.Body)
	s.Require().NoError(err, "failed to delete by query")

	log.Infof("deleted %d documents", len(result.Hits.Hits))
}

// OpenSearchSearchByQuery searches for documents matching the provided query.
func (s *Suite) OpenSearchSearchByQuery(query string, index string) internal.QueryResponse {
	ctx := s.GetContext()
	log := context.GetLogger(*ctx).WithFields(logrus.Fields{
		"query": query,
		"index": index,
	})

	log.Info("deleting by query")
	resp, err := osClient.Search(
		osClient.Search.WithIndex(index),
		osClient.Search.WithBody(strings.NewReader(query)),
	)
	s.Require().NoError(err)
	defer closeSilently(resp.Body)

	result, err := parseElasticSearchResponse[internal.QueryResponse](resp.StatusCode, resp.Body)
	s.Require().NoError(err, "failed to search by query")

	log.Infof("found %d documents", len(result.Hits.Hits))
	return result
}

// OpenSearchSearchCreateDocument creates a new document in the provided index
func (s *Suite) OpenSearchSearchCreateDocument(index string, document map[string]any) {
	ctx := s.GetContext()
	log := context.GetLogger(*ctx).WithFields(logrus.Fields{
		"index": index,
	})

	log.Info("creating document")
	content, err := json.Marshal(document)
	s.Require().NoError(err)

	resp, err := osClient.Index(
		index,
		bytes.NewReader(content),
		osClient.Index.WithContext(ctx),
	)
	s.Require().NoError(err)
	defer closeSilently(resp.Body)

	if resp.IsError() {
		s.T().Fatalf("failed to create document: %v", resp)
	}

	log.Info("document created")
}

// OpenSearchDeleteIndices deletes all esIndices matching the pattern
func (s *Suite) OpenSearchDeleteIndices(pattern string) {
	indices := s.OpenSearchFindIndices(pattern)
	if len(indices) == 0 {
		return
	}
	list := make([]string, 0, len(indices))
	for _, index := range indices {
		list = append(list, index.Name)
	}
	_, err := osClient.Indices.Delete(list)
	s.Require().NoError(err)
}

// openSearchCleanData cleans the data from the opensearch
func (s *Suite) openSearchCleanData() {
	if osClient == nil {
		return
	}

	_, _ = osClient.Indices.Delete(s.osIndices[s.T().Name()])
}
