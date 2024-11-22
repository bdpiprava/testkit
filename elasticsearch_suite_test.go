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

type ElasticSearchSuiteTest struct {
	testkit.Suite
}

func TestElasticSearchSuiteTest(t *testing.T) {
	testkit.Run(t, new(ElasticSearchSuiteTest))
}

func (s *ElasticSearchSuiteTest) Test_RequireElasticSearchClient() {
	s.NotNil(s.RequireElasticSearchClient())
}

func (s *ElasticSearchSuiteTest) Test_CreateIndex() {
	s.ElasticSearchDeleteIndices("test_index_*")
	indexName := fmt.Sprintf("test_index_%d", time.Now().Unix())

	err := s.ElasticSearchCreateIndex(indexName, 1, 1, false, exampleMappings())
	s.Require().NoError(err)

	// Recreate same index should fail
	err = s.ElasticSearchCreateIndex(indexName, 1, 1, false, exampleMappings())
	s.ErrorContains(err, "resource_already_exists_exception")
}

func (s *ElasticSearchSuiteTest) Test_DeleteIndices() {
	s.ElasticSearchDeleteIndices("test_index_*")
	indexName := fmt.Sprintf("test_index_%d", time.Now().Unix())
	s.Require().NoError(s.ElasticSearchCreateIndex(indexName, 1, 1, false, exampleMappings()))
	s.True(s.ElasticSearchIndexExists(indexName))

	// When
	s.ElasticSearchDeleteIndices(indexName)

	// Then
	s.False(s.ElasticSearchIndexExists(indexName))
}

func (s *ElasticSearchSuiteTest) Test_GetIndexSettings() {
	s.ElasticSearchDeleteIndices("test_index_*")
	indexName := fmt.Sprintf("test_index_%d", time.Now().Unix())
	s.Require().NoError(s.ElasticSearchCreateIndex(indexName, 1, 1, false, exampleMappings()))

	// When
	settings := s.ElasticSearchGetIndexSettings(indexName)

	// Then
	s.Equal("1", settings.NumberOfShards)
	s.Equal("1", settings.NumberOfReplicas)
	s.Equal(indexName, settings.ProvidedName)
	s.Nil(settings.Blocks)
}

func (s *ElasticSearchSuiteTest) Test_FindIndices() {
	s.ElasticSearchDeleteIndices("*_test_*")
	expectedIndices := make(search.Indices, 0, 2)
	randomNumber := time.Now().Unix()
	for i := range 2 {
		indexName := fmt.Sprintf("%d_test_%d", randomNumber, randomNumber+int64(i))
		s.Require().NoError(s.ElasticSearchCreateIndex(indexName, 1, 1, false, exampleMappings()))
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
	indices := s.ElasticSearchFindIndices("*_test_*")

	// Then
	s.Len(indices, 2)
	s.True(cmp.Equal(expectedIndices, indices, cmpopts.IgnoreFields(search.Index{}, "UUID")))
}

func (s *ElasticSearchSuiteTest) Test_ElasticSearchSearchByQuery() {
	s.ElasticSearchDeleteIndices("test_search_by_query_*")
	indexName := fmt.Sprintf("test_search_by_query_%d", time.Now().Unix())
	s.Require().NoError(s.ElasticSearchCreateIndex(indexName, 1, 1, false, exampleMappings()))
	s.ElasticSearchSearchCreateDocument(indexName, map[string]any{
		"id":   "1",
		"name": "Bob",
	})
	s.ElasticSearchSearchCreateDocument(indexName, map[string]any{
		"id":   "2",
		"name": "Alice",
	})
	s.ElasticSearchSearchCreateDocument(indexName, map[string]any{
		"id":   "3",
		"name": "Bob",
	})

	s.Eventually(s.hasDocumentCounts(indexName, `{"query": {"query_string": {"query": "*"}}}`, 3))
	s.Eventually(s.hasDocumentCounts(indexName, `{"query": {"bool": {"must":[{"term": {"name": "Bob"}}]}}}`, 2))
	s.Eventually(s.hasDocumentCounts(indexName, `{"query": {"bool": {"must":[{"term": {"id": "1"}}]}}}`, 1))
}

func (s *ElasticSearchSuiteTest) Test_ElasticSearchDeleteByQuery() {
	s.ElasticSearchDeleteIndices("test_delete_by_query_*")
	indexName := fmt.Sprintf("test_delete_by_query_%d", time.Now().Unix())
	s.Require().NoError(s.ElasticSearchCreateIndex(indexName, 1, 1, false, exampleMappings()))
	s.ElasticSearchSearchCreateDocument(indexName, map[string]any{
		"id":   "1",
		"name": "Bob",
	})
	s.ElasticSearchSearchCreateDocument(indexName, map[string]any{
		"id":   "2",
		"name": "Alice",
	})
	s.Eventually(s.hasDocumentCounts(indexName, `{"query": {"query_string": {"query": "*"}}}`, 2))

	// When
	s.ElasticSearchDeleteByQuery(`{"query": {"bool": {"must":[{"term": {"id": "1"}}]}}}`, indexName)

	// Then
	s.Eventually(s.hasDocumentCounts(indexName, `{"query": {"query_string": {"query": "*"}}}`, 1))
}

func (s *ElasticSearchSuiteTest) hasDocumentCounts(index, query string, expectedCount int) (func() bool, time.Duration, time.Duration) {
	return func() bool {
		result := s.ElasticSearchSearchByQuery(query, index)
		return len(result.Hits.Hits) == expectedCount
	}, 5 * time.Second, 200 * time.Millisecond
}
