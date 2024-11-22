package testkit_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/bdpiprava/testkit"
)

type APIMockSuiteTestSuite struct {
	testkit.Suite
}

func TestAPIMockSuiteTestSuite(t *testing.T) {
	testkit.Run(t, new(APIMockSuiteTestSuite))
}

func (s *APIMockSuiteTestSuite) Test_SetupAPIMocksFromFile_ShouldReturnServiceURLs() {
	serviceURLs := s.SetupAPIMocksFromFile("internal/testdata/test-data.yaml", map[string]string{
		"limit": "10",
	})

	s.Require().Len(serviceURLs, 2)
	url1 := "http://localhost:8181/example-service-1/TestAPIMockSuiteTestSuite_Test_SetupAPIMocksFromFile_ShouldReturnServiceURLs"
	url2 := "http://localhost:8181/example-service-2/TestAPIMockSuiteTestSuite_Test_SetupAPIMocksFromFile_ShouldReturnServiceURLs"
	s.Equal(map[string]string{
		"example-service-1": url1,
		"example-service-2": url2,
	}, serviceURLs)
}

func (s *APIMockSuiteTestSuite) Test_SetupAPIMocksFromFile_ShouldSetupMocks() {
	s.CleanAPIMock()
	serviceURLs := s.SetupAPIMocksFromFile("internal/testdata/test-data.yaml", map[string]string{
		"limit": "10",
	})

	req, err := http.NewRequestWithContext(
		s.GetContext(),
		http.MethodGet,
		serviceURLs["example-service-1"]+"/api/v1/example?limit=10&page=1",
		nil,
	)
	s.Require().NoError(err)
	req.Header.Set("Authorization", "Bearer abcd")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	s.Require().NoError(err)
	defer resp.Body.Close()
	s.Equal(http.StatusOK, resp.StatusCode)
	s.Equal("application/json", resp.Header.Get("Content-Type"))

	content, err := io.ReadAll(resp.Body)
	s.Require().NoError(err)

	var body exampleResponse
	err = json.Unmarshal(content, &body)
	s.Require().NoError(err)

	s.Equal("Hello, World!", body.Message)
}

type exampleResponse struct {
	Message string `json:"message"`
}
