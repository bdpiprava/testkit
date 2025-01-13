package testkit

import (
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // postgres driver

	"github.com/bdpiprava/testkit/internal"
)

type psqlDataHolder struct {
	generatedName string
	actualName    string
	db            *sqlx.DB
	helper        *internal.PostgresDB
}

var errDBNotInitiated = fmt.Errorf("database not initiated, must call RequiresPostgresDatabase before using this method")

// RequiresPostgresDatabase is a helper function to get the test database based on configuration
func (s *Suite) RequiresPostgresDatabase(name string) *sqlx.DB {
	var err error
	ctx := s.GetContext()
	postgresDB, err := internal.NewPostgresDB(suiteConfig.PostgresConfig)
	s.Require().NoError(err)

	generatedName := s.generateDatabaseName(name)
	db, err := postgresDB.CreateDatabase(ctx, generatedName, s.Logger())
	s.Require().NoError(err)

	dataHolder := psqlDataHolder{
		generatedName: generatedName,
		actualName:    name,
		helper:        postgresDB,
		db:            db,
	}
	s.postgresDBs[s.T().Name()] = dataHolder

	return db
}

// cleanDatabase delete the database instance
func (s *Suite) cleanDatabase() {
	for _, holder := range s.postgresDBs {
		if holder.db == nil {
			continue
		}

		_ = holder.db.Close()
		_ = holder.helper.Delete(holder.generatedName)
	}
}

// generateName generates a name with the given prefix and a timestamp
func (s *Suite) generateDatabaseName(prefix string) string {
	return strings.ToLower(fmt.Sprintf("%s_%d", prefix, time.Now().UnixMilli()))
}

// PsqlDB returns the database instance for the current test
// if initiated else returns error
func (s *Suite) PsqlDB() (*sqlx.DB, error) {
	if dataHolder, ok := s.postgresDBs[s.T().Name()]; ok {
		return dataHolder.db, nil
	}
	return nil, errDBNotInitiated
}

// PsqlDSN returns the database connection string for the current test db
// if initiated else returns error
func (s *Suite) PsqlDSN() (string, error) {
	dataHolder, ok := s.postgresDBs[s.T().Name()]
	if !ok {
		return "", errDBNotInitiated
	}

	return dataHolder.helper.DSN(dataHolder.generatedName), nil
}

// PsqlDBRecursively returns the database instance starting from current test to parent tests
// if initiated else returns error
// In case of, TestOne -> TestOne/SubTestOne -> TestOne/SubTestOne/SubSubTestOne
// If SubSubTestOne is trying to access the database, it will first check if it has a database
// if not then it will check SubTestOne and then TestOne
func (s *Suite) PsqlDBRecursively() (*sqlx.DB, error) {
	testName := s.T().Name()
	parts := strings.Split(testName, "/")

	for i := len(parts); i >= 0; i-- {
		name := strings.Join(parts[0:i], "/")
		if dataHolder, ok := s.postgresDBs[name]; ok {
			return dataHolder.db, nil
		}
	}

	return nil, errDBNotInitiated
}

// PsqlDSNRecursively returns the database connection string starting from current test to parent tests
// if initiated else returns error
// In case of, TestOne -> TestOne/SubTestOne -> TestOne/SubTestOne/SubSubTestOne
// If SubSubTestOne is trying to access the database, it will first check if it has a database
// if not then it will check SubTestOne and then TestOne
func (s *Suite) PsqlDSNRecursively() (string, error) {
	testName := s.T().Name()
	parts := strings.Split(testName, "/")

	for i := len(parts); i >= 0; i-- {
		name := strings.Join(parts[0:i], "/")
		if dataHolder, ok := s.postgresDBs[name]; ok {
			return dataHolder.helper.DSN(dataHolder.generatedName), nil
		}
	}

	return "", errDBNotInitiated
}
