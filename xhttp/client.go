package xhttp

import (
	"net/http"
	"time"
)

const (
	defaultTimeout = 10 * time.Second
)

// Client is a struct that holds the options and base URL for the client
type Client struct {
	clientOptions ClientOptions
	client        *http.Client
}

// NewClient is a function that returns a new client with the given options and base URL
func NewClient(opts ...ClientOption) *Client {
	cOpts := ClientOptions{
		Timeout: defaultTimeout,
		Headers: http.Header{},
		BaseURL: "",
	}
	for _, opt := range opts {
		opt(&cOpts)
	}

	return &Client{
		clientOptions: cOpts,
		client:        &http.Client{Timeout: cOpts.Timeout},
	}
}

// Execute executes the request and returns the response or an error
func (c Client) Execute(req Request, respType any) (*Response, error) {
	return execute(&c, &req, respType)
}

// WithDefaultTimeout is a function that sets the timeout for the client
func WithDefaultTimeout(timeout time.Duration) ClientOption {
	return func(c *ClientOptions) {
		c.Timeout = timeout
	}
}

// WithDefaultBaseURL is a function that sets the base URL for the client
func WithDefaultBaseURL(baseURL string) ClientOption {
	return func(c *ClientOptions) {
		c.BaseURL = baseURL
	}
}

// WithDefaultHeaders is a function that sets the headers for the client
func WithDefaultHeaders(headers http.Header) ClientOption {
	return func(c *ClientOptions) {
		if headers == nil {
			return
		}

		for k, v := range headers {
			c.Headers[k] = v
		}
	}
}

// WithDefaultHeader is a function that sets the headers for the client
func WithDefaultHeader(key string, values ...string) ClientOption {
	return func(c *ClientOptions) {
		if cur, ok := c.Headers[key]; ok {
			c.Headers[key] = append(cur, values...)
			return
		}
		c.Headers[key] = values
	}
}
