package testkit

import (
	"net/url"
)

const scheme = "postgres"

// Config is the configuration for the postgres database provider
type Config struct {
	Database     string            `yaml:"name"`          // Database of the database
	User         string            `yaml:"user"`          // User of the database
	Password     string            `yaml:"password"`      // Password of the database
	Host         string            `yaml:"host"`          // Host of the database, you can also provide port e.g. localhost:5432
	QueryParams  map[string]string `yaml:"query_params"`  // QueryParams of the database
	FromTemplate string            `yaml:"from_template"` // FromTemplate prepare the database from the template
}

type configRoot struct {
	Postgres Config `yaml:"postgres"`
}

func (c *Config) DSN(name string) string {
	dsn := url.URL{}
	dsn.Scheme = scheme
	dsn.User = url.UserPassword(c.User, c.Password)
	dsn.Host = c.Host
	dsn.Path = name
	params := url.Values{}

	for key, value := range c.QueryParams {
		params.Add(key, value)
	}
	dsn.RawQuery = params.Encode()
	return dsn.String()
}
