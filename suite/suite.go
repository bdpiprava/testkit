package suite

import "github.com/stretchr/testify/suite"

// Suite can store and return the current *testing.T context generated by 'go test'.
type Suite struct {
	suite.Suite
}
