package maps

import (
	"fmt"
	"reflect"
)

const (
	expectedKeyInActualButNotPresent = "Actual: %+v\n\tExpected: %+v\n\tHint: Key '%s' is not present in actual map"
	expectedValueMismatch            = "Actual: %+v\n\tExpected: %+v\n\tHint: Value for key '%s' does not match expected value '%T(%v)' but got '%T(%v)'"
)

// Contains checks if map is subset of another map
//
//	s.AssertMapContains({"x": 1, "y": 2}, {"x": 1}) 										- TRUE
//	s.AssertMapContains({"x": 1, "y": 2}, {"x": 1, "y": 2}) 								- TRUE
//	s.AssertMapContains({"x": 1, "y": 2}, {"x": 2}) 										- FALSE
//	s.AssertMapContains({"x": 1, "y": {"a":"1", "b":"2"}}, {"x": 2, "y": {"a":"1"}}) 		- TRUE
func Contains(actual, expectedSubSet map[string]any) bool {
	if expectedSubSet == nil {
		return true
	}

	if actual == nil {
		return false
	}

	for k, smValue := range expectedSubSet {
		bValue, ok := actual[k]
		if !ok {
			return false
		}

		if smMap, ok := smValue.(map[string]any); ok {
			bMap, ok := bValue.(map[string]any)
			if !ok {
				return false
			}

			if !Contains(bMap, smMap) {
				return false
			}
			continue
		}

		if !reflect.DeepEqual(bValue, smValue) {
			return false
		}
	}

	return true
}

// ContainsWithReason checks if map is subset of another map
//
//	s.AssertMapContains({"x": 1, "y": 2}, {"x": 1}) 										- TRUE
//	s.AssertMapContains({"x": 1, "y": 2}, {"x": 1, "y": 2}) 								- TRUE
//	s.AssertMapContains({"x": 1, "y": 2}, {"x": 2}) 										- FALSE
//	s.AssertMapContains({"x": 1, "y": {"a":"1", "b":"2"}}, {"x": 2, "y": {"a":"1"}}) 		- TRUE
func ContainsWithReason(big, small map[string]any) (bool, string) {
	if small == nil {
		return true, ""
	}

	if big == nil {
		return false, "actual is nil"
	}

	for k, smValue := range small {
		bValue, ok := big[k]
		if !ok {
			return false, fmt.Sprintf(expectedKeyInActualButNotPresent, big, small, k)
		}

		if smMap, ok := smValue.(map[string]any); ok {
			bMap, ok := bValue.(map[string]any)
			if !ok {
				return false, fmt.Sprintf("For key '%s', expected value of type 'map' but got '%T'", k, bValue)
			}

			result, reason := ContainsWithReason(bMap, smMap)
			if !result {
				return false, fmt.Sprintf("For key '%s'\n\t%s", k, reason)
			}
			continue
		}

		if !reflect.DeepEqual(bValue, smValue) {
			return false, fmt.Sprintf(expectedValueMismatch, big, small, k, smValue, smValue, bValue, bValue)
		}
	}
	return true, ""
}
