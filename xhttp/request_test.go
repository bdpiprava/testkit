package xhttp_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/bdpiprava/testkit"
	"github.com/bdpiprava/testkit/xhttp"
)

type serverAPI struct {
	method string
	status int
	body   string
}

type testCase struct {
	name           string
	serverAPI      serverAPI
	wantStatusCode int
	wantResponse   any
	wantErr        string
	wantRawBody    string
}

type RequestTestSuite struct {
	testkit.Suite
}

func TestRequestTestSuite(t *testing.T) {
	testkit.Run(t, new(RequestTestSuite))
}

func (s *RequestTestSuite) TearDownSuite() {
	s.CleanAPIMock()
}

func (s *RequestTestSuite) Test_GET() {
	s.run(getTestCases("GET"), xhttp.GET[map[string]any])
}

func (s *RequestTestSuite) Test_POST() {
	s.run(getTestCases("POST"), xhttp.POST[map[string]any])
}

func (s *RequestTestSuite) Test_DELETE() {
	s.run(getTestCases("DELETE"), xhttp.DELETE[map[string]any])
}

func (s *RequestTestSuite) Test_PUT() {
	s.run(getTestCases("PUT"), xhttp.PUT[map[string]any])
}

func (s *RequestTestSuite) run(testCases []testCase, caller func(opts ...xhttp.RequestOption) (*xhttp.Response, error)) {
	for _, tc := range testCases {
		s.Run(tc.name, func() {
			randomID := uuid.New().String()
			serviceURLs := s.SetupAPIMocksFromFile("../internal/testdata/clients/request.yaml", map[string]string{
				"method":   tc.serverAPI.method,
				"status":   fmt.Sprintf("%d", tc.serverAPI.status),
				"body":     tc.serverAPI.body,
				"randomID": randomID,
			})

			resp, err := caller(
				xhttp.WithBaseURL(serviceURLs["example-service-1"]),
				xhttp.WithPath("/api/v1", randomID),
				xhttp.WithQueryParam("region", "us"),
				xhttp.WithHeader("Authorization", "Bearer abcd"),
				xhttp.WithHeader("Content-Type", "application/json"),
			)

			s.Require().Equal(tc.wantStatusCode, resp.StatusCode)
			s.Require().Equal(tc.wantResponse, resp.Body)
			if tc.wantErr != "" {
				s.Require().Error(err)
				s.EqualError(err, tc.wantErr)
				s.Require().Equal(tc.wantRawBody, string(resp.RawBody))
			}
		})
	}
}

func getTestCases(method string) []testCase {
	method = strings.ToUpper(method)
	return []testCase{
		{
			name:           fmt.Sprintf("%s request with 200 status code", method),
			serverAPI:      serverAPI{method: method, status: 200, body: `{"name": "test"}`},
			wantResponse:   map[string]any{"name": "test"},
			wantStatusCode: 200,
		},
		{
			name:           fmt.Sprintf("%s request with 404 status code", method),
			serverAPI:      serverAPI{method: method, status: 404, body: `{"error": "not found"}`},
			wantResponse:   map[string]any{"error": "not found"},
			wantStatusCode: 404,
		},
		{
			name:           fmt.Sprintf("%s request with 500 status code", method),
			serverAPI:      serverAPI{method: method, status: 500, body: `{"error": "internal server error"}`},
			wantResponse:   map[string]any{"error": "internal server error"},
			wantStatusCode: 500,
		},
		{
			name:           fmt.Sprintf("%s request with invalid json payload", method),
			serverAPI:      serverAPI{method: method, status: 200, body: `not-a-json`},
			wantStatusCode: 200,
			wantErr:        "failed to unmarshal response as type map[string]interface {}: invalid character 'o' in literal null (expecting 'u')",
			wantRawBody:    `not-a-json`,
		},
	}
}
