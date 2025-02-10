package xhttp_test

import (
	"testing"

	"github.com/google/uuid"

	"github.com/bdpiprava/testkit"
	"github.com/bdpiprava/testkit/xhttp"
)

type ClientTestSuite struct {
	testkit.Suite
}

func TestClientTestSuite(t *testing.T) {
	testkit.Run(t, new(ClientTestSuite))
}

func (s *ClientTestSuite) TearDownSuite() {
	s.CleanAPIMock()
}

func (s *ClientTestSuite) Test_Execute() {
	s.Run("should return response", func() {
		// given
		mockURL := s.SetAPIMock(uuid.New().String(), "GET", "/test", 200, `{"key": "value"}`)
		cl := xhttp.NewClient(xhttp.WithDefaultBaseURL(mockURL))

		// when
		req := xhttp.NewRequest("GET", xhttp.WithPath("/test"))
		resp, err := cl.Execute(*req, map[string]any{})

		// then
		s.Require().NoError(err)
		s.Require().Equal(200, resp.StatusCode)
		s.Require().Equal(map[string]any{"key": "value"}, resp.Body)
	})
}
