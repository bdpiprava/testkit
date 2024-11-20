package testkit

import (
	"fmt"
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
func ToWiremockRequest(from Request, dynamicParams map[string]string) *wiremock.StubRule {
	req := wiremock.NewStubRule(from.Method, wiremock.URLEqualTo(from.Path))
	if strings.TrimSpace(from.Body) != "" {
		req = req.WithBodyPattern(wiremock.EqualToJson(from.Body))
	}

	for name, value := range from.QueryParams {
		req = req.WithQueryParam(name, wiremock.Matching(resolveTemplateValue(value, dynamicParams)))
	}

	for name, value := range from.Headers {
		req = req.WithHeader(name, wiremock.Matching(resolveTemplateValue(value, dynamicParams)))
	}
	return req
}

// ToWiremockResponse converts the Response to a wiremock.Response
func ToWiremockResponse(from Response) wiremock.Response {
	resp := wiremock.NewResponse().
		WithJSONBody(from.Body).
		WithStatus(from.Status)

	for name, value := range from.Headers {
		resp.WithHeader(name, value)
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
