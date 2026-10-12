package github

import (
	"reflect"
	"testing"
)

func TestExtractLinkedIssues(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected []int
	}{
		{
			name:     "Single issue Fixes",
			body:     "This PR fixes #42 and enhances performance.",
			expected: []int{42},
		},
		{
			name:     "Multiple issues with varied keywords",
			body:     "Resolves #101, also closes #202 and Closes #101 (duplicate).",
			expected: []int{101, 202},
		},
		{
			name:     "No linked issues",
			body:     "A refactoring without ticket references.",
			expected: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractLinkedIssues(tc.body)
			if len(got) == 0 && len(tc.expected) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tc.expected) {
				t.Errorf("ExtractLinkedIssues(%q) = %v, expected %v", tc.body, got, tc.expected)
			}
		})
	}
}
