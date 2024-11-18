package internal_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/bdpiprava/testkit/internal"
	"github.com/stretchr/testify/assert"
)

func Test_ReadConfig_ViaEnv(t *testing.T) {
	t.Cleanup(unsetEnvVar)
	_ = os.Setenv(internal.EnvDisableConfigCache, "true")
	_ = os.Setenv(internal.EnvConfigLocation, "testdata/config.yml")

	content, err := internal.ReadConfigFile()

	assert.NoError(t, err)
	assert.Len(t, content, 15)
}

func Test_ReadConfig_ViaWorkingDir(t *testing.T) {
	t.Cleanup(unsetEnvVar)
	_ = os.Setenv(internal.EnvDisableConfigCache, "true")
	content, err := internal.ReadConfigFile()

	assert.NoError(t, err)
	assert.Len(t, content, 26)
}

func Test_ReadConfigAs(t *testing.T) {
	t.Cleanup(unsetEnvVar)
	_ = os.Setenv(internal.EnvDisableConfigCache, "true")

	type Result map[string]any

	testCases := []struct {
		name        string
		fileContent string
		want        Result
	}{
		{
			name:        "should load empty yaml file",
			fileContent: "",
		},
		{
			name: "should load yaml file with values",
			fileContent: `
key:
  sub-key1: sub-value1
  sub-key2: sub-value2
key-2: value-2
`,
			want: map[string]any{
				"key": Result{
					"sub-key1": "sub-value1",
					"sub-key2": "sub-value2",
				},
				"key-2": "value-2",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			location := filepath.Join(t.TempDir(), "config.yml")
			err := os.WriteFile(location, []byte(tc.fileContent), 0644)
			assert.NoError(t, err)
			_ = os.Setenv(internal.EnvConfigLocation, location)

			result, err := internal.ReadConfigAs[Result]()

			assert.NoError(t, err)
			assert.Equal(t, tc.want, result)
		})
	}
}

func unsetEnvVar() {
	_ = os.Unsetenv(internal.EnvConfigLocation)
	_ = os.Unsetenv(internal.EnvDisableConfigCache)
}
