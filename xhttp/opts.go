package xhttp

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

// BasicAuth is a struct that holds the username and password for basic authentication
type BasicAuth struct {
	Username string
	Password string
}

// ClientOptions is a struct that holds the options for the client
type ClientOptions struct {
	BaseURL   string
	Headers   http.Header
	BasicAuth BasicAuth
	Timeout   time.Duration
}

// ClientOption is a function that takes a pointer to Options and modifies it
type ClientOption func(client *ClientOptions)

// RequestOptions is a struct that holds the options for the request
type RequestOptions struct {
	Method      string
	BaseURL     string
	Headers     http.Header
	QueryParams url.Values
	Body        io.Reader
	BasicAuth   BasicAuth
	Path        string
	Timeout     time.Duration
}

// RequestOption is a function that takes a pointer to Options and modifies it
type RequestOption func(*RequestOptions)
