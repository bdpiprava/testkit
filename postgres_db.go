package testkit

import (
	"fmt"
	"net/url"

	"github.com/bdpiprava/testkit/context"
	"github.com/bdpiprava/testkit/internal"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	scheme                = "postgres"
	rootDatabase          = "postgres"
	databaseExistsQuery   = `SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = '%v')`
	createTemplateDBQuery = `CREATE DATABASE %v WITH IS_TEMPLATE=TRUE`
	createDBQuery         = `CREATE DATABASE %v`
)

// PostgresConfig is the configuration for the postgres database provider
type PostgresConfig struct {
	Database     string            `yaml:"name"`          // Database of the database
	User         string            `yaml:"user"`          // User of the database
	Password     string            `yaml:"password"`      // Password of the database
	Host         string            `yaml:"host"`          // Host of the database, you can also provide port e.g. localhost:5432
	QueryParams  map[string]string `yaml:"query_params"`  // QueryParams of the database
	FromTemplate string            `yaml:"from_template"` // FromTemplate prepare the database from the template
}

// psqlConfigRoot represents the postgres config root
type psqlConfigRoot struct {
	Postgres PostgresConfig `yaml:"postgres"`
}

// PostgresDB helper to do operation on postgres database
type PostgresDB struct {
	config PostgresConfig // PostgresConfig configuration for the postgres database
}

// NewPostgresDB returns new instance of PostgresDB
func NewPostgresDB() (*PostgresDB, error) {
	config, err := internal.ReadConfigAs[psqlConfigRoot]()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read config")
	}

	return &PostgresDB{
		config: config.Postgres,
	}, nil
}

// DSN returns the DSN with given database name
func (p *PostgresDB) DSN(name string) string {
	return p.config.DSN(name)
}

// connect returns a connection to the database
// ensure that connection is established by making a ping request
func (p *PostgresDB) connect(name string) (*sqlx.DB, error) {
	dsn := p.DSN(name)
	root, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		return nil, errors.Wrapf(err, "[%s] failed to connect to database", name)
	}

	err = root.Ping()
	if err != nil {
		return nil, errors.Wrapf(err, "[%s] ping database failed", name)
	}

	return root, nil
}

// delete deletes a database with the given name
func (p *PostgresDB) delete(name string) error {
	root, err := p.connect(rootDatabase)
	if err != nil {
		return err
	}
	defer closeSilently(root)

	_, err = root.Exec(fmt.Sprintf(`DROP DATABASE %v`, name))
	return err
}

// deleteTemplateDB removes the template database
func (p *PostgresDB) deleteTemplateDB(name string) error {
	root, err := p.connect(rootDatabase)
	if err != nil {
		return err
	}
	defer closeSilently(root)

	_, err = root.Exec(fmt.Sprintf("ALTER DATABASE %s is_template FALSE", name))
	if err != nil {
		return err
	}

	_, err = root.Exec(fmt.Sprintf(`DROP DATABASE %v`, name))
	return err
}

// createDatabase creates a new target database from the template database
func (p *PostgresDB) createDatabase(ctx context.Context, targetName string) (*sqlx.DB, error) {
	log := context.GetLogger(ctx).WithFields(logrus.Fields{
		"func":     "createDatabase",
		"target":   targetName,
		"template": p.config.FromTemplate,
	})

	root, err := p.connect(rootDatabase)
	if err != nil {
		return nil, err
	}
	defer closeSilently(root)

	// return on error or if the database already exists
	if exists, err := p.exists(root, targetName); err != nil || exists {
		log.Info("Database already exists")
		return p.connect(targetName)
	}

	if len(p.config.FromTemplate) > 0 {
		log.Info("Creating new database from template")
		_, err = root.ExecContext(ctx, fmt.Sprintf(createFromTemplateQuery, targetName, p.config.FromTemplate))
		if err != nil {
			return nil, errors.Wrap(err, "failed to create database from template")
		}

		return p.connect(targetName)
	}

	log.Info("Creating new database from scratch")
	_, err = root.ExecContext(ctx, fmt.Sprintf(createDBQuery, targetName))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create database from scratch")
	}

	return p.connect(targetName)
}

// exists checks if the database exists
func (p *PostgresDB) exists(db *sqlx.DB, name string) (bool, error) {
	var exists bool
	err := db.Get(&exists, fmt.Sprintf(databaseExistsQuery, name))
	return exists, err
}

// DSN returns the DSN with given database name
func (p *PostgresConfig) DSN(name string) string {
	dsn := url.URL{}
	dsn.Scheme = scheme
	dsn.User = url.UserPassword(p.User, p.Password)
	dsn.Host = p.Host
	dsn.Path = name
	params := url.Values{}

	for key, value := range p.QueryParams {
		params.Add(key, value)
	}
	dsn.RawQuery = params.Encode()
	return dsn.String()
}
