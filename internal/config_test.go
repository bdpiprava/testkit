package internal_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bdpiprava/testkit/internal"
)

func Test_ReadConfig_ProjectRoot(t *testing.T) {
	unsetEnvVar()
	t.Cleanup(unsetEnvVar)
	_ = os.Setenv(internal.EnvDisableConfigCache, "true")

	content, err := internal.ReadConfigFile()

	require.NoError(t, err)
	require.NotEmpty(t, content)
}

func Test_ReadConfig_ViaEnv(t *testing.T) {
	t.Cleanup(unsetEnvVar)
	_ = os.Setenv(internal.EnvDisableConfigCache, "true")
	_ = os.Setenv(internal.EnvConfigLocation, "testdata/config.yml")

	content, err := internal.ReadConfigFile()

	require.NoError(t, err)
	require.Len(t, content, 15)
}

func Test_ReadConfig_ViaWorkingDir(t *testing.T) {
	t.Cleanup(unsetEnvVar)
	_ = os.Setenv(internal.EnvDisableConfigCache, "true")
	content, err := internal.ReadConfigFile()

	require.NoError(t, err)
	require.NotEmpty(t, content)
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
			err := os.WriteFile(location, []byte(tc.fileContent), 0600)
			require.NoError(t, err)
			_ = os.Setenv(internal.EnvConfigLocation, location)

			result, err := internal.ReadConfigAs[Result]()

			require.NoError(t, err)
			require.Equal(t, tc.want, result)
		})
	}
}

func unsetEnvVar() {
	_ = os.Unsetenv(internal.EnvConfigLocation)
	_ = os.Unsetenv(internal.EnvDisableConfigCache)
}
