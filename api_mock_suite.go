package testkit

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var testNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// SetupAPIMocksFromFile set up the services mock from a file and return the URLs
func (s *Suite) SetupAPIMocksFromFile(file string, dynamicParams map[string]string) map[string]string {
	root, err := readFile(file)
	s.NoError(err)

	serviceURLs := make(map[string]string)

	for name, paths := range root {
		testPath := filepath.Join(name, testNameSanitizer.ReplaceAllString(s.T().Name(), "_"))
		serviceURLs[name], err = url.JoinPath(suiteConfig.APIMockConfig.Address, testPath)
		s.NoError(err)

		for _, path := range paths {
			path.Request.Path = filepath.Join(testPath, path.Request.Path)
			err = wiremockClient.StubFor(path.Request.ToWiremockRequest(dynamicParams).
				WillReturnResponse(path.Response.ToWiremockResponse()).
				AtPriority(1))

			s.NoError(err)
		}
	}

	return serviceURLs
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
