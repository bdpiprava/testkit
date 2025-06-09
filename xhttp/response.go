package xhttp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/pkg/errors"
)

const successStatusCode = 299

// Response is a response struct that holds the status code, body and raw body
type Response struct {
	Status     string
	header     http.Header
	StatusCode int
	Body       any
	RawBody    []byte
}

// newResponse is a function that creates a new response
func newResponse(httpResp *http.Response, bType any) (*Response, error) {
	defer httpResp.Body.Close()

	if bType == nil {
		return nil, fmt.Errorf("unsupported type: %T", bType)
	}

	bodyBytes, _ := io.ReadAll(httpResp.Body)
	response := &Response{
		header:     httpResp.Header,
		Status:     httpResp.Status,
		StatusCode: httpResp.StatusCode,
		RawBody:    bodyBytes,
	}

	if httpResp.StatusCode > successStatusCode {
		response.Body = tryParsingErrorResponse(bodyBytes)
		return response, nil
	}

	err := json.Unmarshal(bodyBytes, &bType)
	if err != nil {
		return response, errors.Wrapf(err, "failed to unmarshal response as type %T", bType)
	}

	response.Body = bType
	return response, nil
}

// tryParsingErrorResponse is a function that tries to parse the error response as JSON object or returns the raw body
func tryParsingErrorResponse(contentBytes []byte) any {
	parsedBody := make(map[string]any)
	if json.Unmarshal(contentBytes, &parsedBody) != nil {
		return string(contentBytes)
	}
	return parsedBody
}
