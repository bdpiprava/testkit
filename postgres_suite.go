package testkit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres driver

	"github.com/bdpiprava/testkit/internal"
)

type ctxKey string

const (
	keyDatabaseName ctxKey = "database_name"
	keyDatabase     ctxKey = "database"
)

// RequiresPostgresDatabase is a helper function to get the test database based on configuration
func (s *Suite) RequiresPostgresDatabase(name string) *sqlx.DB {
	var err error
	ctx := s.GetContext()
	s.postgresDB, err = internal.NewPostgresDB(suiteConfig.PostgresConfig)
	s.Require().NoError(err)

	generatedName := s.generateDatabaseName(name)
	db, err := s.postgresDB.CreateDatabase(ctx, generatedName, s.Logger())
	s.Require().NoError(err)
	s.ctx = context.WithValue(ctx, keyDatabaseName, generatedName)
	s.ctx = context.WithValue(s.ctx, keyDatabase, db)
	return db
}

// cleanDatabase delete the database instance
func (s *Suite) cleanDatabase() {
	ctx := s.GetContext()
	if db, ok := ctx.Value(keyDatabase).(*sqlx.DB); ok {
		if db == nil {
			return
		}
		_ = db.Close()
		_ = s.postgresDB.Delete(ctx.Value(keyDatabaseName).(string))
	}
}

// generateName generates a name with the given prefix and a timestamp
func (s *Suite) generateDatabaseName(prefix string) string {
	return strings.ToLower(fmt.Sprintf("%s_%d", prefix, time.Now().UnixMilli()))
}

// PsqlDB returns the database instance for the test if initiated or returns error
func (s *Suite) PsqlDB() (*sqlx.DB, error) {
	ctx := s.GetContext()
	if db, ok := ctx.Value(keyDatabase).(*sqlx.DB); ok {
		return db, nil
	}
	return nil, fmt.Errorf("database not initiated, must call RequiresPostgresDatabase before using this method")
}
