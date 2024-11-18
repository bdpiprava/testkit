package testkit_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/bdpiprava/testkit"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/suite"
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
	testkit.ElasticSearchSuite
}

func TestElasticSearchSuiteTest(t *testing.T) {
	suite.Run(t, new(ElasticSearchSuiteTest))
}

func (s *ElasticSearchSuiteTest) Test_CreateIndex() {
	s.DeleteIndices("test_index_*")
	indexName := fmt.Sprintf("test_index_%d", time.Now().Unix())

	err := s.CreateIndex(indexName, 1, 1, false, exampleMappings())
	s.Require().NoError(err)

	// Recreate same index should fail
	err = s.CreateIndex(indexName, 1, 1, false, exampleMappings())
	s.ErrorContains(err, "resource_already_exists_exception")
}

func (s *ElasticSearchSuiteTest) Test_DeleteIndices() {
	s.DeleteIndices("test_index_*")
	indexName := fmt.Sprintf("test_index_%d", time.Now().Unix())
	s.Require().NoError(s.CreateIndex(indexName, 1, 1, false, exampleMappings()))
	s.True(s.IndexExists(indexName))

	// When
	s.DeleteIndices(indexName)

	// Then
	s.False(s.IndexExists(indexName))
}

func (s *ElasticSearchSuiteTest) Test_GetIndexSettings() {
	s.DeleteIndices("test_index_*")
	indexName := fmt.Sprintf("test_index_%d", time.Now().Unix())
	s.Require().NoError(s.CreateIndex(indexName, 1, 1, false, exampleMappings()))

	// When
	settings := s.GetIndexSettings(indexName)

	// Then
	s.Equal("1", settings.NumberOfShards)
	s.Equal("1", settings.NumberOfReplicas)
	s.Equal(indexName, settings.ProvidedName)
	s.Nil(settings.Blocks)
}

func (s *ElasticSearchSuiteTest) Test_FindIndices() {
	s.DeleteIndices("*_test_*")
	expectedIndices := make(testkit.Indices, 0, 2)
	randomNumber := time.Now().Unix()
	for i := 0; i < 2; i++ {
		indexName := fmt.Sprintf("%d_test_%d", randomNumber, randomNumber+int64(i))
		s.Require().NoError(s.CreateIndex(indexName, 1, 1, false, exampleMappings()))
		expectedIndices = append(expectedIndices, testkit.Index{
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
	indices := s.FindIndices("*_test_*")

	// Then
	s.Len(indices, 2)
	s.True(cmp.Equal(expectedIndices, indices, cmpopts.IgnoreFields(testkit.Index{}, "UUID")))
}
