package testkit

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/wiremock/go-wiremock"
)

// {{(.*?)}} is the template matcher
var templateMatcher = regexp.MustCompile(`{{(.*?)}}`)

type mockRoot map[string][]Path

// Path is the request and response information
type Path struct {
	Request  Request  `yaml:"request" json:"request"`
	Response Response `yaml:"response" json:"response"`
}

// Request is the request information
type Request struct {
	Method      string            `yaml:"method" json:"method"`
	Path        string            `yaml:"path" json:"path"`
	Body        string            `yaml:"body" json:"body"`
	Headers     map[string]string `yaml:"headers" json:"headers"`
	QueryParams map[string]string `yaml:"queryParams" json:"queryParams"`
}

// Response is the response information
type Response struct {
	Status  int64             `yaml:"status" json:"status"`
	Body    string            `yaml:"body" json:"body"`
	Headers map[string]string `yaml:"headers" json:"headers"`
}

// ToWiremockRequest converts the Request to a wiremock.StubRule
func (r *Request) ToWiremockRequest(dynamicParams map[string]string) *wiremock.StubRule {
	query := url.Values{}
	for name, value := range r.QueryParams {
		query.Set(name, resolveTemplateValue(value, dynamicParams))
	}

	req := wiremock.NewStubRule(r.Method, wiremock.URLEqualTo(fmt.Sprintf("/%s?%s", r.Path, query.Encode())))
	if strings.TrimSpace(r.Body) != "" {
		req = req.WithBodyPattern(wiremock.EqualToJson(r.Body))
	}

	for name, value := range query {
		req = req.WithQueryParam(name, wiremock.Matching(value[0]))
	}

	for name, value := range r.Headers {
		req = req.WithHeader(name, wiremock.Matching(resolveTemplateValue(value, dynamicParams)))
	}
	return req
}

// ToWiremockResponse converts the Response to a wiremock.Response
func (r *Response) ToWiremockResponse() wiremock.Response {
	resp := wiremock.NewResponse().
		WithBody(r.Body).
		WithStatus(r.Status)

	for name, value := range r.Headers {
		resp = resp.WithHeader(name, value)
	}
	return resp
}

// resolveTemplateValue resolves the template value
func resolveTemplateValue(str string, params map[string]string) string {
	if templateMatcher.MatchString(str) {
		matches := templateMatcher.FindStringSubmatch(str)
		for _, match := range matches {
			tmpl := fmt.Sprintf("{{%s}}", match)
			str = strings.ReplaceAll(str, tmpl, params[strings.TrimSpace(match)])
		}
	}
	return str
}