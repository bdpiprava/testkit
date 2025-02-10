package xhttp

import (
	"net/http"
	"net/url"
)

var supportedMethods = map[string]bool{
	http.MethodGet:     true,
	http.MethodPost:    true,
	http.MethodPut:     true,
	http.MethodPatch:   true,
	http.MethodDelete:  true,
	http.MethodConnect: true,
	http.MethodOptions: true,
	http.MethodTrace:   true,
	http.MethodHead:    true,
}

var defaultClient = &Client{client: http.DefaultClient}

// Request is a request struct
type Request struct {
	opts []RequestOption
}

// NewRequest is a function that returns a new request with the given options
func NewRequest(method string, opts ...RequestOption) *Request {
	opts = append(opts, func(c *RequestOptions) {
		c.Method = method
	})
	return &Request{opts: opts}
}

// WithBaseURL is a function that sets the base URL for the request
func WithBaseURL(baseURL string) RequestOption {
	return func(c *RequestOptions) {
		c.BaseURL = baseURL
	}
}

// WithPath is a function that sets the base URL for the request
func WithPath(paths ...string) RequestOption {
	return func(c *RequestOptions) {
		u := &url.URL{}
		c.Path = u.JoinPath(paths...).String()
	}
}

// WithHeaders is a function that sets the headers for the request
func WithHeaders(headers http.Header) RequestOption {
	return func(c *RequestOptions) {
		if headers == nil {
			return
		}

		for k, v := range headers {
			c.Headers[k] = v
		}
	}
}

// WithHeader is a function that sets the headers for the request
func WithHeader(key string, values ...string) RequestOption {
	return func(c *RequestOptions) {
		if cur, ok := c.Headers[key]; ok {
			c.Headers[key] = append(cur, values...)
			return
		}
		c.Headers[key] = values
	}
}

// WithQueryParams is a function that sets the query parameters for the request
func WithQueryParams(params url.Values) RequestOption {
	return func(c *RequestOptions) {
		if params == nil {
			return
		}

		for k, v := range params {
			c.QueryParams[k] = v
		}
	}
}

// WithQueryParam is a function that sets the query parameters for the request
func WithQueryParam(key string, values ...string) RequestOption {
	return func(c *RequestOptions) {
		if cur, ok := c.QueryParams[key]; ok {
			c.QueryParams[key] = append(cur, values...)
			return
		}
		c.QueryParams[key] = values
	}
}

// GET is a function that sends a GET request
func GET[T any](opts ...RequestOption) (*Response, error) {
	req := NewRequest(http.MethodGet, opts...)
	return defaultClient.Execute(*req, *(new(T)))
}

// POST is a function that sends a POST request
func POST[T any](opts ...RequestOption) (*Response, error) {
	req := NewRequest(http.MethodPost, opts...)
	return defaultClient.Execute(*req, *(new(T)))
}

// PUT is a function that sends a PUT request
func PUT[T any](opts ...RequestOption) (*Response, error) {
	req := NewRequest(http.MethodPut, opts...)
	return defaultClient.Execute(*req, *(new(T)))
}

// DELETE is a function that sends a DELETE request
func DELETE[T any](opts ...RequestOption) (*Response, error) {
	req := NewRequest(http.MethodDelete, opts...)
	return defaultClient.Execute(*req, *(new(T)))
}
