package testkit

import (
	"errors"
	"fmt"
	"time"

	_ "github.com/lib/pq"

	"github.com/bdpiprava/testkit/context"
	"github.com/jmoiron/sqlx"
)

const (
	keyDatabaseName context.Key = "database_name"
	keyDatabase     context.Key = "database"
)

func init() {
	_, err := InitialiseDatabase()
	if err != nil && !errors.Is(err, ErrMissingGoMigrateConfig) {
		panic(err)
	}
}

// PostgresSuite is a suite that provides tooling for postgres integration tests
type PostgresSuite struct {
	context.ContextSuite
	postgresDB *PostgresDB
}

// RequiresPostgresDatabase is a helper function to get the test database based on configuration
func (s *PostgresSuite) RequiresPostgresDatabase(name string) *sqlx.DB {
	err := s.Initialize(s.T().Name())
	s.Require().NoError(err)

	ctx := s.GetContext(s.T().Name())
	s.postgresDB, err = NewPostgresDB()
	s.Require().NoError(err)

	generatedName := s.generateName(name)
	ctx.SetData(keyDatabaseName, generatedName)
	db, err := s.postgresDB.createDatabase(*ctx, name)
	s.Require().NoError(err)
	ctx.SetData(keyDatabase, db)
	return db
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
		_ = s.postgresDB.delete(ctx.Value(keyDatabaseName).(string))
	}
}

// generateName generates a name with the given prefix and a timestamp
func (s *PostgresSuite) generateName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixMilli())
}
