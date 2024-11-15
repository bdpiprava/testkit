package testkit

import (
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/bdpiprava/testkit/context"
	"github.com/bdpiprava/testkit/internal"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	rootDatabase            = "postgres"
	databaseExistsQuery     = `SELECT EXISTS(SELECT datname FROM pg_catalog.pg_database WHERE datname = '%v')`
	createTemplateDBQuery   = `CREATE DATABASE %v WITH IS_TEMPLATE=TRUE`
	createDBQuery           = `CREATE DATABASE %v`
	createFromTemplateQuery = `CREATE DATABASE %v WITH TEMPLATE '%v' OWNER ccs`

	keyDatabaseName context.Key = "database_name"
	keyDatabase     context.Key = "database"
)

// PostgresSuite is a suite that provides tooling for postgres integration tests
type PostgresSuite struct {
	context.ContextSuite
	testKitConfig Config
}

// RequiresPostgresDatabase is a helper function to get the test database based on configuration
func (s *PostgresSuite) RequiresPostgresDatabase(name string) (*sqlx.DB, error) {
	if err := s.Initialize(s.T().Name()); err != nil {
		return nil, err
	}

	ctx := s.GetContext(s.T().Name())
	config, err := internal.ReadConfigAs[configRoot]()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read config")
	}

	s.testKitConfig = config.Postgres
	generatedName := s.generateName(name)
	ctx.SetData(keyDatabaseName, generatedName)
	db, err := s.createDatabase(*ctx, name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create db")
	}
	ctx.SetData(keyDatabase, db)
	return db, nil
}

// TearDownSuite perform the cleanup of the database
func (s *PostgresSuite) TearDownSuite() {
	defer s.CleanDatabase()
}

// CleanDatabase delete the database instance
func (s *PostgresSuite) CleanDatabase() {
	ctx := s.GetContext(s.T().Name())
	if db, ok := ctx.Value(keyDatabase).(*sqlx.DB); ok {
		if db == nil {
			return
		}
		_ = db.Close()
		_ = s.testKitConfig.delete(ctx.Value(keyDatabaseName).(string))
	}
}

// generateName generates a name with the given prefix and a timestamp
func (s *PostgresSuite) generateName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixMilli())
}

// createDatabase creates a new target database from the template database
func (s *PostgresSuite) createDatabase(ctx context.Context, targetName string) (*sqlx.DB, error) {
	log := context.GetLogger(ctx).WithFields(logrus.Fields{
		"func":     "createDatabase",
		"target":   targetName,
		"template": s.testKitConfig.FromTemplate,
	})

	root, err := s.testKitConfig.connect(rootDatabase)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	// return on error or if the database already exists
	if exists, err := exists(root, targetName); err != nil || exists {
		log.Info("Database already exists")
		return s.testKitConfig.connect(targetName)
	}

	if len(s.testKitConfig.FromTemplate) > 0 {
		log.Info("Creating new database from template")
		_, err = root.ExecContext(ctx, fmt.Sprintf(createFromTemplateQuery, targetName, s.testKitConfig.FromTemplate))
		if err != nil {
			return nil, errors.Wrap(err, "failed to create database from template")
		}

		return s.testKitConfig.connect(targetName)
	}

	log.Info("Creating new database from scratch")
	_, err = root.ExecContext(ctx, fmt.Sprintf(createDBQuery, targetName))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create database from scratch")
	}

	return s.testKitConfig.connect(targetName)
}

// connect returns a connection to the database
// ensure that connection is established by making a ping request
func (c *Config) connect(name string) (*sqlx.DB, error) {
	dsn := c.DSN(name)
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
func (c *Config) delete(name string) error {
	root, err := c.connect("postgres")
	if err != nil {
		return err
	}
	defer root.Close()

	_, err = root.Exec(fmt.Sprintf(`DROP DATABASE %v`, name))
	return err
}

// exists checks if the database exists
func exists(db *sqlx.DB, name string) (bool, error) {
	var exists bool
	err := db.Get(&exists, fmt.Sprintf(databaseExistsQuery, name))
	return exists, err
}
