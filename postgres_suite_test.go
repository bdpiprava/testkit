package testkit_test

import (
	"testing"

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
