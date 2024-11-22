package testkit

import (
	"flag"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"runtime/debug"
	"testing"
)

var allTestsFilter = func(_, _ string) (bool, error) { return true, nil }
var matchMethod = flag.String("testkit.m", "", "regular expression to select tests of the testify suite to run")

// Run runs the suite
func Run(t *testing.T, suite TestingSuite) {
	defer recoverAndFailOnPanic(t)
	suite.SetT(t)
	suite.SetS(suite)

	var suiteSetupDone bool
	tests := make([]testing.InternalTest, 0)
	methodFinder := reflect.TypeOf(suite)
	for i := range methodFinder.NumMethod() {
		method := methodFinder.Method(i)
		ok, err := methodFilter(method.Name)
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "testkit: invalid regexp for -m: %s\n", err)

			// we need to exit with a non-zero status to indicate that the tests failed
			//nolint:gocritic
			os.Exit(1)
		}

		if !ok {
			continue
		}

		if !suiteSetupDone {
			if setup, ok := suite.(SetupSuite); ok {
				setup.SetupSuite()
			}
			suiteSetupDone = true
		}

		test := testing.InternalTest{
			Name: method.Name,
			F: func(t *testing.T) {
				parentT := suite.T()
				suite.SetT(t)
				defer recoverAndFailOnPanic(t)
				defer func() {
					t.Helper()
					r := recover()
					if tearDownTestSuite, ok := suite.(TearDownTest); ok {
						tearDownTestSuite.TearDownTest()
					}

					suite.SetT(parentT)
					failOnPanic(t, r)
				}()

				if setupTestSuite, ok := suite.(SetupTest); ok {
					setupTestSuite.SetupTest()
				}
				method.Func.Call([]reflect.Value{reflect.ValueOf(suite)})
			},
		}
		tests = append(tests, test)
	}

	if suiteSetupDone {
		defer func() {
			if tearDownSuite, ok := suite.(TearDownSuite); ok {
				tearDownSuite.TearDownSuite()
			}
		}()
	}

	runTests(t, tests)
}
func recoverAndFailOnPanic(t *testing.T) {
	t.Helper()
	r := recover()
	failOnPanic(t, r)
}

func failOnPanic(t *testing.T, r interface{}) {
	t.Helper()
	if r != nil {
		t.Errorf("test panicked: %v\n%s", r, debug.Stack())
		t.FailNow()
	}
}

// Filtering method according to set regular expression
// specified command-line argument -m
func methodFilter(name string) (bool, error) {
	if ok, _ := regexp.MatchString("^Test", name); !ok {
		return false, nil
	}
	return regexp.MatchString(*matchMethod, name)
}

func runTests(t testing.TB, tests []testing.InternalTest) {
	if len(tests) == 0 {
		t.Log("warning: no tests to run")
		return
	}

	r, ok := t.(runner)
	if !ok {
		// backwards compatibility with Go 1.6 and below
		if !testing.RunTests(allTestsFilter, tests) {
			t.Fail()
		}
		return
	}

	for _, test := range tests {
		r.Run(test.Name, test.F)
	}
}

type runner interface {
	Run(name string, f func(t *testing.T)) bool
}
