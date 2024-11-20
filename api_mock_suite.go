package testkit

import (
	"net/url"
	"os"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/suite"
	"github.com/wiremock/go-wiremock"
	"gopkg.in/yaml.v3"

	"github.com/bdpiprava/testkit/internal"
)

var testNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9]+`)

// apiMockRoot is the root configuration for the API mock
type apiMockRoot struct {
	APIMockConfig *APIMockConfig `yaml:"api-mock"`
}

// APIMockConfig is the configuration for the API mock
type APIMockConfig struct {
	Address string `yaml:"address" default:"localhost:8080"`
}

var wiremockClient *wiremock.Client
var address = "http://localhost:8080"

func init() {
	config, err := internal.ReadConfigAs[apiMockRoot]()
	if err != nil {
		panic(err)
	}

	if config.APIMockConfig == nil {
		wiremockClient = wiremock.NewClient("http://localhost:8080")
		return
	}

	address = config.APIMockConfig.Address
	wiremockClient = wiremock.NewClient(address)
}

// APIMockSuite is a suite that provides tooling for API mock tests
type APIMockSuite struct {
	suite.Suite
}

// FromFile set up the services mock from a file and return the URLs
func (s *APIMockSuite) FromFile(file string, dynamicParams map[string]string) map[string]string {
	root, err := readFile(file)
	s.Require().NoError(err)

	serviceURLs := make(map[string]string)

	for name, paths := range root {
		testPath := filepath.Join(testNameSanitizer.ReplaceAllString(s.T().Name(), "_"), name)
		serviceURLs[name], err = url.JoinPath(address, testPath)
		s.Require().NoError(err)

		for _, path := range paths {
			path.Request.Path = filepath.Join(testPath, path.Request.Path)
			err = wiremockClient.StubFor(ToWiremockRequest(path.Request, dynamicParams).
				WillReturnResponse(ToWiremockResponse(path.Response)).
				AtPriority(1))

			s.Require().NoError(err)
		}
	}

	return serviceURLs
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
