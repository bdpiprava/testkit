package testkit

import (
	"fmt"
	"reflect"

	"github.com/stretchr/testify/assert"
)

const mapContainsFailTemplate = `Map does not contains:
Actual: %#v
Subset: %#v
Hint: For key '%s', subset value of type '%s' does not match value of type '%s' in actual
`

// AssertMapContains checks if map is subset of another map
//
//	s.AssertMapContains({"x": 1, "y": 2}, {"x": 1}) 										- Pass
//	s.AssertMapContains({"x": 1, "y": 2}, {"x": 1, "y": 2}) 								- Pass
//	s.AssertMapContains({"x": 1, "y": 2}, {"x": 2}) 										- Fail
//	s.AssertMapContains({"x": 1, "y": {"a":"1", "b":"2"}}, {"x": 2, "y": {"a":"1"}}) 		- Pass
func (s *Suite) AssertMapContains(big, small interface{}) bool {
	smallKind := reflect.TypeOf(small).Kind()
	bigKind := reflect.TypeOf(big).Kind()
	if smallKind != reflect.Map {
		return s.Fail(fmt.Sprintf("%q has an unsupported type %s", small, smallKind))
	}

	if bigKind != reflect.Map {
		return s.Fail(fmt.Sprintf("%q has an unsupported type %s", big, bigKind))
	}

	if small == nil {
		return true
	}

	if big == nil {
		return false
	}

	bigMap := reflect.ValueOf(big)
	smallMap := reflect.ValueOf(small)
	for _, k := range smallMap.MapKeys() {
		smValue := smallMap.MapIndex(k)
		bValue := bigMap.MapIndex(k)

		smKind := reflect.TypeOf(smValue.Interface()).Kind()
		bKind := reflect.TypeOf(bValue.Interface()).Kind()
		if !bValue.IsValid() {
			return s.Fail(fmt.Sprintf(mapContainsFailTemplate, bigMap, smallMap, k, smKind.String(), bKind.String()))
		}

		if smKind == reflect.Map {
			if !s.AssertMapContains(bValue.Interface(), smValue.Interface()) {
				return s.Fail(fmt.Sprintf(mapContainsFailTemplate, bigMap, smallMap, k, smKind.String(), bKind.String()))
			}
			continue
		}

		if !assert.ObjectsAreEqual(smValue.Interface(), bValue.Interface()) {
			return s.Fail(fmt.Sprintf(mapContainsFailTemplate, bigMap, smallMap, k, smKind.String(), bKind.String()))
		}
	}
	return true
}
