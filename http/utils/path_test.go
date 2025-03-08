package utils_test

import (
	"testing"

	"gitlab.com/iglou.eu/goulc/http/utils"
)

func TestPathFormatting(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty path",
			input:    "",
			expected: "/",
		},
		{
			name:     "root path",
			input:    "/",
			expected: "/",
		},
		{
			name:     "path without leading slash",
			input:    "test/path",
			expected: "/test/path",
		},
		{
			name:     "path with trailing slash",
			input:    "/test/path/",
			expected: "/test/path",
		},
		{
			name:     "path with both leading and trailing slash",
			input:    "/test/path/",
			expected: "/test/path",
		},
		{
			name:     "single directory without slashes",
			input:    "test",
			expected: "/test",
		},
		{
			name:     "multiple nested directories",
			input:    "/a/b/c/d",
			expected: "/a/b/c/d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := utils.PathFormatting(tt.input)
			if got != tt.expected {
				t.Errorf("PathFormatting(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
