package internal

import (
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // postgres driver
	_ "github.com/golang-migrate/migrate/v4/source/file"       // file driver
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const createFromTemplateQuery = `CREATE DATABASE %v WITH TEMPLATE '%v' OWNER ccs`

// ErrMissingGoMigrateConfig ...
var ErrMissingGoMigrateConfig = errors.New("missing go-migrate config")

// InitialiseDatabase create a new database when go migrate is configured
func InitialiseDatabase(config SuiteConfig, log logrus.FieldLogger) (*sqlx.DB, error) {
	if config.GoMigrateConfig == nil {
		log.Warn("missing go-migrate config in the config file")
		return nil, ErrMissingGoMigrateConfig
	}

	cfg := config.GoMigrateConfig
	log = log.WithFields(logrus.Fields{
		"database":       cfg.DatabaseName,
		"template":       cfg.IsTemplate,
		"migration_path": cfg.MigrationPath,
		"fresh":          cfg.Fresh,
	})

	postgresDB, err := NewPostgresDB(config.PostgresConfig)
	if err != nil {
		log.WithError(err).Error("go migrate failed as it failed to initialise the postgres helper")
		return nil, errors.Wrap(err, "go migrate failed")
	}

	root, err := postgresDB.connect(rootDatabase)
	if err != nil {
		log.WithError(err).Errorf("failed to connect to %s database", rootDatabase)
		return nil, err
	}
	defer closeSilently(root)

	exists, err := postgresDB.exists(root, cfg.DatabaseName)
	if err != nil {
		log.WithError(err).Errorf("failed to check database exist for %s", cfg.DatabaseName)
		return nil, err
	}

	if exists {
		if !cfg.Fresh {
			log.Debugf("template database '%s' already exists, returning...", cfg.DatabaseName)
			return postgresDB.connect(cfg.DatabaseName)
		}

		log.Info("exist but requested fresh database, hence deleting the existing database")
		err := postgresDB.deleteTemplateDB(cfg.DatabaseName)
		if err != nil {
			log.WithError(err).Error("failed to delete database")
			return nil, err
		}
	}

	createDatabaseQuery := getCreateDatabaseQuery(cfg.DatabaseName, cfg.IsTemplate)
	_, err = root.Exec(createDatabaseQuery)
	if err != nil {
		log.WithError(err).Error("failed to create database")
		return nil, errors.Wrap(err, "failed to create database")
	}

	migrator, err := migrate.New(fmt.Sprintf("file://%s", cfg.MigrationPath), postgresDB.DSN(cfg.DatabaseName))
	if err != nil {
		log.WithError(err).Error("failed to initialize migrations")
		return nil, errors.Wrap(err, "failed to initialize migrations")
	}

	if err := migrator.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		log.WithError(err).Error("failed to apply migrations")
		return nil, errors.Wrap(err, "failed to apply migrations")
	}

	return postgresDB.connect(cfg.DatabaseName)
}

func getCreateDatabaseQuery(name string, template bool) string {
	if template {
		return fmt.Sprintf(createTemplateDBQuery, name)
	}
	return fmt.Sprintf(createDBQuery, name)
}
