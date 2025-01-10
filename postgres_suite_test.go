package testkit_test

import (
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/bdpiprava/testkit"
)

type DatabaseIntegrationTestSuite struct {
	testkit.Suite
}

func TestDatabaseIntegrationTestSuite(t *testing.T) {
	testkit.Run(t, new(DatabaseIntegrationTestSuite))
}

func (s *DatabaseIntegrationTestSuite) TestSuite_RequiresPostgresDatabase() {
	db := s.RequiresPostgresDatabase("test")

	var version string
	err := db.Get(&version, "SELECT VERSION()")
	s.Require().NoError(err)

	s.Require().NotEmpty(version)
	s.Contains(version, "PostgreSQL")
}

func (s *DatabaseIntegrationTestSuite) TestSuite_PsqlDB_Success() {
	db := s.RequiresPostgresDatabase("PsqlDB_Success")
	version := s.getVersion(db)
	s.NotEmpty(version)

	got, gotErr := s.PsqlDB()

	s.NoError(gotErr)
	s.Equal(version, s.getVersion(got))
}

func (s *DatabaseIntegrationTestSuite) TestSuite_PsqlDB_Failure() {
	got, gotErr := s.PsqlDB()

	s.Nil(got)
	s.EqualError(gotErr, "database not initiated, must call RequiresPostgresDatabase before using this method")
}

func (s *DatabaseIntegrationTestSuite) getVersion(db *sqlx.DB) string {
	var version string
	err := db.Get(&version, "SELECT VERSION()")
	s.Require().NoError(err)
	return version
}
