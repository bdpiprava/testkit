package testkit

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"github.com/wiremock/go-wiremock"
	"gopkg.in/yaml.v3"
)

var testNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// SetupAPIMocksFromFile set up the services mock from a file and return the URLs
func (s *Suite) SetupAPIMocksFromFile(file string, dynamicParams map[string]string) map[string]string {
	root, err := readFile(file)
	s.NoError(err)

	if dynamicParams == nil {
		dynamicParams = make(map[string]string)
	}

	serviceURLs := make(map[string]string)
	for name, paths := range root {
		testPath := filepath.Join(name, testNameSanitizer.ReplaceAllString(s.T().Name(), "_"))
		serviceURLs[name], err = url.JoinPath(suiteConfig.APIMockConfig.Address, testPath)
		s.NoError(err)

		for _, path := range paths {
			path.Request.Path = filepath.Join(testPath, path.Request.Path)
			err = wiremockClient.StubFor(path.Request.ToWiremockRequest(dynamicParams).
				WillReturnResponse(path.Response.ToWiremockResponse(dynamicParams)).
				AtPriority(1))

			s.NoError(err)
		}
	}

	return serviceURLs
}

// SetAPIMock sets the wiremock server with the given method, path, status and body
func (s *Suite) SetAPIMock(namespace, method, path string, status int, body string) string {
	namespace = testNameSanitizer.ReplaceAllString(namespace, "_")
	stubRule := wiremock.NewStubRule(method, wiremock.URLMatching(filepath.Join("/", namespace, path))).
		WillReturnResponse(wiremock.NewResponse().WithStatus(int64(status)).WithBody(body)).
		AtPriority(1)

	s.Require().NoError(wiremockClient.StubFor(stubRule))

	mockURL, err := url.JoinPath(suiteConfig.APIMockConfig.Address, namespace)
	s.Require().NoError(err)

	return mockURL
}

// CleanAPIMock resets the wiremock server
func (s *Suite) CleanAPIMock() {
	err := wiremockClient.Reset()
	s.Require().NoError(err)
}

// readFile reads the config file and unmarshal it into the given type
func readFile(path string) (mockRoot, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config mockRoot
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal api mock data from file: %v", path)
	}

	return config, nil
}
