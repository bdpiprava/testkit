package testkit_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bdpiprava/testkit"
)

type APIMockSuiteTestSuite struct {
	testkit.APIMockSuite
}

func TestAPIMockSuiteTestSuite(t *testing.T) {
	suite.Run(t, new(APIMockSuiteTestSuite))
}

func (s *APIMockSuiteTestSuite) TestFromFile() {
	serviceURLs := s.FromFile("internal/testdata/test-data.yaml", map[string]string{
		"limit": "10",
	})
	s.Require().Len(serviceURLs, 2)
	s.Equal(map[string]string{
		"example-service-1": "http://localhost:8181/TestAPIMockSuiteTestSuite_TestFromFile/example-service-1",
		"example-service-2": "http://localhost:8181/TestAPIMockSuiteTestSuite_TestFromFile/example-service-2",
	}, serviceURLs)
}
