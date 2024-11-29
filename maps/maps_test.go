package maps_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bdpiprava/testkit/maps"
)

type containsTestCases struct {
	name       string
	actual     map[string]any
	expected   map[string]any
	want       bool
	wantReason string
}

func Test_Contains(t *testing.T) {
	testCases := getContainsTestCases()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := maps.Contains(tc.actual, tc.expected)

			assert.Equal(t, tc.want, got)
		})
	}
}

func Test_ContainsWithReason(t *testing.T) {
	testCases := getContainsTestCases()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, gotReason := maps.ContainsWithReason(tc.actual, tc.expected)

			assert.Equal(t, tc.want, got)
			assert.Equal(t, tc.wantReason, gotReason)
		})
	}
}

func getContainsTestCases() []containsTestCases {
	return []containsTestCases{
		{
			name:       "actual is nil",
			actual:     nil,
			expected:   map[string]any{},
			want:       false,
			wantReason: "actual is nil",
		},
		{
			name:       "expected is nil",
			actual:     map[string]any{},
			expected:   nil,
			want:       true,
			wantReason: "",
		},
		{
			name:       "actual contains expected",
			actual:     map[string]any{"a": 1, "b": 2},
			expected:   map[string]any{"a": 1},
			want:       true,
			wantReason: "",
		},
		{
			name:       "actual does not contain expected",
			actual:     map[string]any{"a": 1, "b": 2},
			expected:   map[string]any{"a": 2},
			want:       false,
			wantReason: "Actual: map[a:1 b:2]\n\tExpected: map[a:2]\n\tHint: Value for key 'a' does not match expected value 'int(2)' but got 'int(1)'",
		},
		{
			name:       "actual contain expected but type mismatch",
			actual:     map[string]any{"a": float32(1), "b": 2},
			expected:   map[string]any{"a": int32(1)},
			want:       false,
			wantReason: "Actual: map[a:1 b:2]\n\tExpected: map[a:1]\n\tHint: Value for key 'a' does not match expected value 'int32(1)' but got 'float32(1)'",
		},
		{
			name:       "nested map contains expected",
			actual:     map[string]any{"a": map[string]any{"b": 1}},
			expected:   map[string]any{"a": map[string]any{"b": 1}},
			want:       true,
			wantReason: "",
		},
		{
			name:       "nested map does not contain expected",
			actual:     map[string]any{"a": map[string]any{"b": 1}},
			expected:   map[string]any{"a": map[string]any{"b": 2}},
			want:       false,
			wantReason: "For key 'a'\n\tActual: map[b:1]\n\tExpected: map[b:2]\n\tHint: Value for key 'b' does not match expected value 'int(2)' but got 'int(1)'",
		},
	}
}
