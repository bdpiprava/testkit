package testkit

import (
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres driver

	"github.com/bdpiprava/testkit/context"
	"github.com/bdpiprava/testkit/internal"
)

const (
	keyDatabaseName context.Key = "database_name"
	keyDatabase     context.Key = "database"
)

// RequiresPostgresDatabase is a helper function to get the test database based on configuration
func (s *Suite) RequiresPostgresDatabase(name string) *sqlx.DB {
	var err error
	ctx := s.GetContext(s.T().Name())
	s.postgresDB, err = internal.NewPostgresDB(suiteConfig.PostgresConfig)
	s.Require().NoError(err)

	generatedName := s.generateDatabaseName(name)
	ctx.SetData(keyDatabaseName, generatedName)
	db, err := s.postgresDB.CreateDatabase(*ctx, name)
	s.Require().NoError(err)
	ctx.SetData(keyDatabase, db)
	return db
}

// cleanDatabase delete the database instance
func (s *Suite) cleanDatabase() {
	ctx := s.GetContext(s.T().Name())
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
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixMilli())
}
