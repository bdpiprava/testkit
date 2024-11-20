package internal

// SuiteConfig is the configuration for the test suite
type SuiteConfig struct {
	LogLevel        string               `yaml:"log_level"`     // LogLevel is the log level
	PostgresConfig  PostgresConfig       `yaml:"postgres"`      // PostgresConfig configuration for the postgres database
	ElasticSearch   *ElasticSearchConfig `yaml:"elasticsearch"` // ElasticSearchConfig configuration for the elastic search client
	OpenSearch      *ElasticSearchConfig `yaml:"opensearch"`    // OpenSearch configuration for the elastic search client
	GoMigrateConfig *GoMigrateConfig     `yaml:"go-migrate"`    // GoMigrateConfig config for go migrate
	APIMockConfig   *APIMockConfig       `yaml:"api-mock"`      // APIMockConfig configuration for the API mock
}

// PostgresConfig is the configuration for the postgres database provider
type PostgresConfig struct {
	Database     string            `yaml:"name"`          // Database of the database
	User         string            `yaml:"user"`          // User of the database
	Password     string            `yaml:"password"`      // Password of the database
	Host         string            `yaml:"host"`          // Host of the database e.g. localhost:5432
	QueryParams  map[string]string `yaml:"query_params"`  // QueryParams of the database
	FromTemplate string            `yaml:"from_template"` // FromTemplate prepare the database from the template
}

// ElasticSearchConfig is the configuration for the elastic search client
type ElasticSearchConfig struct {
	Addresses string `yaml:"addresses"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
}

// GoMigrateConfig represent the go migrate config
type GoMigrateConfig struct {
	MigrationPath string `yaml:"migration_path"` // MigrationPath path the migration files
	DatabaseName  string `yaml:"database_name"`  // DatabaseName name of the database
	IsTemplate    bool   `yaml:"is_template"`    // IsTemplate create database as template
	Fresh         bool   `yaml:"fresh"`          // Fresh recreate if one already exists
}

// APIMockConfig is the configuration for the API mock
type APIMockConfig struct {
	Address string `yaml:"address" default:"localhost:8080"`
}
