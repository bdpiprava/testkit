package testkit_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"

	"github.com/bdpiprava/testkit"
	"github.com/bdpiprava/testkit/search"
)

var exampleMappings = func() map[string]any {
	return map[string]any{
		"id": map[string]any{
			"type": "keyword",
		},
		"name": map[string]any{
			"type": "keyword",
		},
	}
}
var createIndexSettings = search.CreateIndexSettings{
	NumberOfShards:    1,
	NumberOfReplicas:  1,
	MappingProperties: exampleMappings(),
}

type OpenSearchSuiteTest struct {
	testkit.Suite
}

func TestOpenSearchSuiteTest(t *testing.T) {
	testkit.Run(t, new(OpenSearchSuiteTest))
}

func (s *OpenSearchSuiteTest) Test_SearchClient() {
	testCases := []struct {
		name   string
		client testkit.SearchClient
	}{
		{
			name:   "ElasticSearch",
			client: s.RequireElasticSearch(),
		},
		{
			name:   "OpenSearch",
			client: s.RequireOpenSearch(),
		},
	}

	for _, tc := range testCases {
		client := tc.client

		s.Run(tc.name+"#CreateIndex", func() {
			s.Require().NoError(client.DeleteIndices("test_index_*"))
			indexName := fmt.Sprintf("test_index_%d", time.Now().Unix())

			err := client.CreateIndex(indexName, createIndexSettings)
			s.Require().NoError(err)

			// Recreate same index should fail
			err = client.CreateIndex(indexName, createIndexSettings)
			s.ErrorContains(err, "resource_already_exists_exception")
		})

		s.Run(tc.name+"#DeleteIndices", func() {
			s.Require().NoError(client.DeleteIndices("test_index_*"))
			indexName := fmt.Sprintf("test_index_%d", time.Now().Unix())
			s.Require().NoError(client.CreateIndex(indexName, createIndexSettings))
			s.True(client.IndexExists(indexName))

			// When
			s.Require().NoError(client.DeleteIndices(indexName))

			// Then
			s.False(client.IndexExists(indexName))
		})

		s.Run(tc.name+"#GetIndexSettings", func() {
			s.Require().NoError(client.DeleteIndices("test_index_*"))
			indexName := fmt.Sprintf("test_index_%d", time.Now().Unix())
			s.Require().NoError(client.CreateIndex(indexName, createIndexSettings))

			// When
			settings, err := client.GetIndexSettings(indexName)
			s.Require().NoError(err)

			// Then
			s.Equal("1", settings.NumberOfShards)
			s.Equal("1", settings.NumberOfReplicas)
			s.Equal(indexName, settings.ProvidedName)
			s.Nil(settings.Blocks)
		})

		s.Run(tc.name+"#FindIndices", func() {
			s.Require().NoError(client.DeleteIndices("*_test_*"))
			expectedIndices := make(search.Indices, 0, 2)
			randomNumber := time.Now().Unix()
			for i := range 2 {
				indexName := fmt.Sprintf("%d_test_%d", randomNumber, randomNumber+int64(i))
				s.Require().NoError(client.CreateIndex(indexName, createIndexSettings))
				expectedIndices = append(expectedIndices, search.Index{
					Name:         indexName,
					Pri:          "1",
					Rep:          "1",
					DocsCount:    "0",
					DocsDeleted:  "0",
					StoreSize:    "208b",
					PriStoreSize: "208b",
					Status:       "open",
					Health:       "yellow",
				})
			}

			// When
			indices, err := client.FindIndices("*_test_*")
			s.Require().NoError(err)

			// Then
			s.Len(indices, 2)
			s.True(cmp.Equal(expectedIndices, indices, cmpopts.IgnoreFields(search.Index{}, "UUID")))
		})

		s.Run(tc.name+"#SearchByQuery", func() {
			s.Require().NoError(client.DeleteIndices("test_search_by_query_*"))
			indexName := fmt.Sprintf("test_search_by_query_%d", time.Now().Unix())
			s.Require().NoError(client.CreateIndex(indexName, createIndexSettings))
			s.Require().NoError(client.CreateDocument(indexName, map[string]any{
				"id":   "1",
				"name": "Bob",
			}))
			s.Require().NoError(client.CreateDocument(indexName, map[string]any{
				"id":   "2",
				"name": "Alice",
			}))
			s.Require().NoError(client.CreateDocument(indexName, map[string]any{
				"id":   "3",
				"name": "Bob",
			}))

			s.Eventually(s.hasDocumentCounts(client, indexName, `{"query": {"query_string": {"query": "*"}}}`, 3))
			s.Eventually(s.hasDocumentCounts(client, indexName, `{"query": {"bool": {"must":[{"term": {"name": "Bob"}}]}}}`, 2))
			s.Eventually(s.hasDocumentCounts(client, indexName, `{"query": {"bool": {"must":[{"term": {"id": "1"}}]}}}`, 1))
		})

		s.Run(tc.name+"#DeleteByQuery", func() {
			s.Require().NoError(client.DeleteIndices("test_delete_by_query_*"))
			indexName := fmt.Sprintf("test_delete_by_query_%d", time.Now().Unix())
			s.Require().NoError(client.CreateIndex(indexName, createIndexSettings))
			s.Require().NoError(client.CreateDocument(indexName, map[string]any{
				"id":   "1",
				"name": "Bob",
			}))
			s.Require().NoError(client.CreateDocument(indexName, map[string]any{
				"id":   "2",
				"name": "Alice",
			}))
			s.Eventually(s.hasDocumentCounts(client, indexName, `{"query": {"query_string": {"query": "*"}}}`, 2))

			// When
			s.Require().NoError(client.DeleteByQuery([]string{indexName}, `{"query": {"bool": {"must":[{"term": {"id": "1"}}]}}}`))

			// Then
			s.Eventually(s.hasDocumentCounts(client, indexName, `{"query": {"query_string": {"query": "*"}}}`, 1))
		})
	}
}

func (s *OpenSearchSuiteTest) hasDocumentCounts(
	client testkit.SearchClient,
	index,
	query string,
	expectedCount int,
) (func() bool, time.Duration, time.Duration) {
	return func() bool {
		result, err := client.SearchByQuery(index, query)
		s.Require().NoError(err)
		return len(result.Hits.Hits) == expectedCount
	}, 5 * time.Second, 200 * time.Millisecond
}
