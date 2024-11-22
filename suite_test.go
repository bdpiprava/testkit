package testkit_test

import (
	"testing"

	"github.com/bdpiprava/testkit"
)

type FooSuite struct {
	testkit.Suite
}

func (s *FooSuite) SetupSuite() {
	println("Setting up FooSuite")
}

type BarSuite struct {
	FooSuite
}

func (s *BarSuite) SetupSuite() {
	println("Setting up BarSuite")
}

type SuiteTestSuite struct {
	BarSuite
}

func TestSuiteTestSuite(t *testing.T) {
	testkit.Run(t, new(SuiteTestSuite))
}

func (s *SuiteTestSuite) SetupSuite() {
	println("Setting up SuiteTestSuite")
	s.RequireOpenSearchClient()
	s.RequireElasticSearchClient()
	s.RequiresPostgresDatabase("test")
	s.RequiresKafka("test")
}

func (s *SuiteTestSuite) TestExampleTest() {
	println("Running ExampleTest")
	//s.RequireOpenSearchClient()
	//s.RequireElasticSearchClient()
	//s.RequiresPostgresDatabase("test")
	//s.RequiresKafka("test")
}
