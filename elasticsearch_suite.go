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

	"github.com/elastic/go-elasticsearch/v7/esapi"

	"github.com/bdpiprava/testkit/internal"
)

// CreateIndex creates a new index
func (s *Suite) CreateIndex(
	index string,
	numberOfShards,
	numberOfReplicas int,
	dynamic bool,
	properties map[string]any,
) error {
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

// IndexExists checks if the index exists
func (s *Suite) IndexExists(name string) bool {
	indices := s.FindIndices(name)
	for _, index := range indices {
		if strings.ToLower(index.Name) == strings.ToLower(name) {
			return true
		}
	}
	return false
}

// CloseIndices closes the indices
func (s *Suite) CloseIndices(indices ...string) {
	ctx := s.GetContext()
	_, err := esClient.Indices.Close(indices, esClient.Indices.Close.WithContext(ctx))
	s.Require().NoError(err)
}

// FindIndices returns matching indices sorted by name
func (s *Suite) FindIndices(pattern string) internal.Indices {
	ctx := s.GetContext()
	resp, err := esClient.Cat.Indices(
		esClient.Cat.Indices.WithContext(ctx),
		esClient.Cat.Indices.WithIndex(pattern),
		esClient.Cat.Indices.WithFormat("json"),
	)
	s.Require().NoError(err)
	defer closeSilently(resp.Body)

	if resp.StatusCode == http.StatusNotFound {
		return make(internal.Indices, 0)
	}

	if resp.IsError() {
		s.T().Fatalf("failed to get all indices: %v", resp)
	}

	respBytes, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	var result internal.Indices
	err = json.Unmarshal(respBytes, &result)
	s.Require().NoError(err)

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})

	return result
}

// GetIndexSettings returns the settings for the given index
func (s *Suite) GetIndexSettings(index string) internal.IndexSetting {
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

// DeleteIndices deletes all indices matching the pattern
func (s *Suite) DeleteIndices(pattern string) {
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

// cleanElasticSearchData cleans the data from the elasticsearch
func (s *Suite) cleanElasticSearchData() {
	_, _ = esClient.Indices.Delete(s.indices[s.T().Name()])
}

// EventuallyBlockStatus waits until the block status is the expected one
func (s *Suite) EventuallyBlockStatus(indexName string, status string, timeout, interval time.Duration) {
	s.Eventually(s.checkBlockStatusFn(indexName, status), timeout, interval)
}

func (s *Suite) checkBlockStatusFn(indexName string, status string) func() bool {
	return func() bool {
		return s.GetIndexSettings(indexName).Blocks.Write == status
	}
}

func closeSilently(closable io.Closer) {
	_ = closable.Close()
}
