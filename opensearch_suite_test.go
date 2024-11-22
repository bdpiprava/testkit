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

type OpenSearchSuiteTest struct {
	testkit.Suite
}

func TestOpenSearchSuiteTest(t *testing.T) {
	testkit.Run(t, new(OpenSearchSuiteTest))
}

func (s *OpenSearchSuiteTest) Test_RequireOpenSearchClient() {
	s.NotNil(s.RequireOpenSearchClient())
}

func (s *OpenSearchSuiteTest) Test_CreateIndex() {
	s.OpenSearchDeleteIndices("test_index_*")
	indexName := fmt.Sprintf("test_index_%d", time.Now().Unix())

	err := s.OpenSearchCreateIndex(indexName, 1, 1, false, exampleMappings())
	s.Require().NoError(err)

	// Recreate same index should fail
	err = s.OpenSearchCreateIndex(indexName, 1, 1, false, exampleMappings())
	s.ErrorContains(err, "resource_already_exists_exception")
}

func (s *OpenSearchSuiteTest) Test_DeleteIndices() {
	s.OpenSearchDeleteIndices("test_delete_indices_*")
	indexName := fmt.Sprintf("test_delete_indices_%d", time.Now().Unix())
	s.Require().NoError(s.OpenSearchCreateIndex(indexName, 1, 1, false, exampleMappings()))
	s.True(s.OpenSearchIndexExists(indexName))

	// When
	s.OpenSearchDeleteIndices(indexName)

	// Then
	s.False(s.OpenSearchIndexExists(indexName))
}

func (s *OpenSearchSuiteTest) Test_GetIndexSettings() {
	s.OpenSearchDeleteIndices("test_get_index_settings_*")
	indexName := fmt.Sprintf("test_get_index_settings_%d", time.Now().Unix())
	s.Require().NoError(s.OpenSearchCreateIndex(indexName, 1, 1, false, exampleMappings()))

	// When
	settings := s.OpenSearchGetIndexSettings(indexName)

	// Then
	s.Equal("1", settings.NumberOfShards)
	s.Equal("1", settings.NumberOfReplicas)
	s.Equal(indexName, settings.ProvidedName)
	s.Nil(settings.Blocks)
}

func (s *OpenSearchSuiteTest) Test_FindIndices() {
	s.OpenSearchDeleteIndices("*_test_find_indices_*")
	expectedIndices := make(search.Indices, 0, 2)
	randomNumber := time.Now().Unix()
	for i := range 2 {
		indexName := fmt.Sprintf("%d_test_find_indices_%d", randomNumber, randomNumber+int64(i))
		s.Require().NoError(s.OpenSearchCreateIndex(indexName, 1, 1, false, exampleMappings()))
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
	indices := s.OpenSearchFindIndices("*_test_find_indices_*")

	// Then
	s.True(cmp.Equal(expectedIndices, indices, cmpopts.IgnoreFields(search.Index{}, "UUID")))
}

func (s *OpenSearchSuiteTest) Test_OpenSearchSearchByQuery() {
	s.OpenSearchDeleteIndices("test_search_by_query_*")
	indexName := fmt.Sprintf("test_search_by_query_%d", time.Now().Unix())
	s.Require().NoError(s.OpenSearchCreateIndex(indexName, 1, 1, false, exampleMappings()))
	s.OpenSearchSearchCreateDocument(indexName, map[string]any{
		"id":   "1",
		"name": "Bob",
	})
	s.OpenSearchSearchCreateDocument(indexName, map[string]any{
		"id":   "2",
		"name": "Alice",
	})
	s.OpenSearchSearchCreateDocument(indexName, map[string]any{
		"id":   "3",
		"name": "Bob",
	})

	s.Eventually(s.hasDocumentCounts(indexName, `{"query": {"query_string": {"query": "*"}}}`, 3))
	s.Eventually(s.hasDocumentCounts(indexName, `{"query": {"bool": {"must":[{"term": {"name": "Bob"}}]}}}`, 2))
	s.Eventually(s.hasDocumentCounts(indexName, `{"query": {"bool": {"must":[{"term": {"id": "1"}}]}}}`, 1))
}

func (s *OpenSearchSuiteTest) Test_OpenSearchDeleteByQuery() {
	s.OpenSearchDeleteIndices("test_delete_by_query_*")
	indexName := fmt.Sprintf("test_delete_by_query_%d", time.Now().Unix())
	s.Require().NoError(s.OpenSearchCreateIndex(indexName, 1, 1, false, exampleMappings()))
	s.OpenSearchSearchCreateDocument(indexName, map[string]any{
		"id":   "1",
		"name": "Bob",
	})
	s.OpenSearchSearchCreateDocument(indexName, map[string]any{
		"id":   "2",
		"name": "Alice",
	})
	s.Eventually(s.hasDocumentCounts(indexName, `{"query": {"query_string": {"query": "*"}}}`, 2))

	// When
	s.OpenSearchDeleteByQuery(`{"query": {"bool": {"must":[{"term": {"id": "1"}}]}}}`, indexName)

	// Then
	s.Eventually(s.hasDocumentCounts(indexName, `{"query": {"query_string": {"query": "*"}}}`, 1))
}

func (s *OpenSearchSuiteTest) hasDocumentCounts(index, query string, expectedCount int) (func() bool, time.Duration, time.Duration) {
	return func() bool {
		result := s.OpenSearchSearchByQuery(query, index)
		return len(result.Hits.Hits) == expectedCount
	}, 5 * time.Second, 200 * time.Millisecond
}
