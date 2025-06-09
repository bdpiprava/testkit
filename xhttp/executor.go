package xhttp

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
)

// execute is a function that executes the request with given client and returns the response
func execute(client *Client, request *Request, respType any) (*Response, error) {
	if respType == nil {
		return nil, errors.New("response type cannot be nil")
	}

	opts := buildOpts(client.clientOptions, request)
	req, err := buildRequest(opts)
	if err != nil {
		return nil, err
	}

	log.Default().Println("Executing request:", req.Method, req.URL.String())
	resp, err := client.client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute request")
	}

	return newResponse(resp, respType)
}

// buildRequest is a function that builds the request from the given options
func buildRequest(opts RequestOptions) (*http.Request, error) {
	if _, ok := supportedMethods[strings.ToUpper(opts.Method)]; !ok {
		return nil, errors.Errorf("unsupported method: %s", opts.Method)
	}

	req, err := http.NewRequest(opts.Method, opts.BaseURL, opts.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	req = req.WithContext(opts.Context)
	req.URL = req.URL.JoinPath(opts.Path)
	req.Header = opts.Headers
	req.URL.RawQuery = opts.QueryParams.Encode()

	return req, nil
}

// buildOpts is a function that builds the request options
func buildOpts(clientOpts ClientOptions, request *Request) RequestOptions {
	opts := RequestOptions{
		Headers:     http.Header{},
		BaseURL:     clientOpts.BaseURL,
		Timeout:     clientOpts.Timeout,
		Method:      http.MethodGet,
		QueryParams: url.Values{},
		Context:     context.Background(),
	}

	if clientOpts.Headers != nil {
		for k, v := range clientOpts.Headers {
			opts.Headers[k] = v
		}
	}

	for _, opt := range request.opts {
		opt(&opts)
	}
	return opts
}
