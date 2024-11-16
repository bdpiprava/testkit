package testkit_test

import (
	"testing"

	"github.com/bdpiprava/testkit"
	"github.com/stretchr/testify/suite"
)

type DatabaseIntegrationTestSuite struct {
	testkit.PostgresSuite
}

func TestDatabaseIntegrationTestSuite(t *testing.T) {
	suite.Run(t, new(DatabaseIntegrationTestSuite))
}

func (s *DatabaseIntegrationTestSuite) TestSuite_RequiresPostgresDatabase() {
	db := s.RequiresPostgresDatabase("test")

	var version string
	err := db.Get(&version, "SELECT VERSION()")
	s.Require().NoError(err)

	s.Require().NotEmpty(version)
	s.Contains(version, "PostgreSQL")
}
