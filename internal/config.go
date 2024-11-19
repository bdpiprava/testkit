package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

var configFileMatcher = regexp.MustCompile(`(?i).testkit.config.(yaml|yml)$`)

const (
	EnvConfigLocation     = "TESTKIT_CONFIG_LOCATION"
	EnvDisableConfigCache = "DISABLE_CONFIG_CACHE"
)

// cache is a struct to hold the content of the file and the path
type cache struct {
	content      []byte
	path         string
	loadedViaEnv bool
	err          error
}

var configCache *cache

// ReadConfigAs reads the config file and unmarshal it into the given type
func ReadConfigAs[T any]() (T, error) {
	var config T
	content, err := ReadConfigFile()
	if err != nil {
		return config, err
	}

	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return config, errors.Wrapf(err, "failed to unmarshal config from file %s", configCache.path)
	}

	return config, nil
}

// ReadConfigFile read the config file as byte array
// 1. read the file from environment variable TESTKIT_CONFIG_LOCATION, if set
// 2. read the file from the current working directory
func ReadConfigFile() ([]byte, error) {
	_, cacheDisabled := os.LookupEnv(EnvDisableConfigCache)
	// If the config file is already loaded, return the content
	if !cacheDisabled && configCache != nil && len(configCache.content) > 0 {
		return configCache.content, configCache.err
	}

	location := strings.TrimSpace(os.Getenv(EnvConfigLocation))
	if location != "" {
		configCache = readConfigFile(location, true)
		return configCache.content, configCache.err
	}

	wd, err := os.Getwd()
	if err != nil {
		configCache = &cache{err: err}
		return nil, err
	}

	location, err = locateConfigFile(wd)
	if err != nil {
		configCache = &cache{err: err}
		return nil, err
	}

	configCache = readConfigFile(location, false)
	return configCache.content, configCache.err
}

// locateConfigFile finds the config file in the current directory or its parent
func locateConfigFile(workingDir string) (string, error) {
	// start from it
	currentDir := filepath.Clean(workingDir)
	info, err := os.Stat(currentDir)
	if err != nil {
		return "", err
	}

	if !info.IsDir() {
		currentDir = filepath.Dir(currentDir)
	}

	for {
		path, err := containsConfigFile(currentDir)
		if err != nil {
			return "", err
		}

		if path != "" {
			return path, nil
		}

		// go up one level
		parent := filepath.Dir(currentDir)
		if currentDir == parent || parent == "" {
			break
		}
		currentDir = parent
	}

	return "", fmt.Errorf("config file not found")
}

// readConfigFile reads the content of the file and returns the cache struct
func readConfigFile(path string, loadedViaEnv bool) *cache {
	content, err := os.ReadFile(path)
	return &cache{
		content:      content,
		err:          err,
		path:         path,
		loadedViaEnv: loadedViaEnv,
	}
}

// containsConfigFile returns file name if the directory contains a config file
func containsConfigFile(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if configFileMatcher.MatchString(entry.Name()) {
			return strings.TrimSpace(filepath.Join(dir, entry.Name())), nil
		}
	}

	return "", nil
}
