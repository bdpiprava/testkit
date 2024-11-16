package testkit_test

import (
	"testing"

	"github.com/bdpiprava/testkit"
)

func TestConfig_BuildConnectionString(t *testing.T) {
	testCases := []struct {
		name   string
		config testkit.PostgresConfig
		want   string
	}{
		{
			name: "should build connection string with default values",
			config: testkit.PostgresConfig{
				Database: "postgres",
				Host:     "localhost",
				User:     "postgres",
				Password: "password",
				QueryParams: map[string]string{
					"sslmode":         "disable",
					"connect_timeout": "10",
				},
			},
			want: "postgres://postgres:password@localhost/test?connect_timeout=10&sslmode=disable",
		},
		{
			name: "should build connection string with host and port",
			config: testkit.PostgresConfig{
				Database: "xyz",
				Host:     "localhost:5432",
				User:     "postgres",
				Password: "password",
				QueryParams: map[string]string{
					"sslmode": "disable",
				},
			},
			want: "postgres://postgres:password@localhost:5432/test?sslmode=disable",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.DSN("test")
			if got != tt.want {
				t.Errorf("DSN() got = %v, want %v", got, tt.want)
			}
		})
	}
}
