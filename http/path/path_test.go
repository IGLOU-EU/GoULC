package path_test

import (
	"testing"

	"gitlab.com/iglou.eu/goulc/http/path"
)

func TestFormat(t *testing.T) {
	tests := []struct {
		name string
		give string
		want string
	}{
		{
			name: "empty path",
			give: "",
			want: "/",
		},
		{
			name: "root path",
			give: "/",
			want: "/",
		},
		{
			name: "path without leading slash",
			give: "test/path",
			want: "/test/path",
		},
		{
			name: "path with trailing slash",
			give: "/test/path/",
			want: "/test/path",
		},
		{
			name: "path with multiple trailing slashes",
			give: "path//",
			want: "/path",
		},
		{
			name: "path with interior double slash",
			give: "/a//b",
			want: "/a/b",
		},
		{
			name: "path with dot segments",
			give: "/a/./b/../c",
			want: "/a/c",
		},
		{
			name: "path escaping above root",
			give: "/../a",
			want: "/a",
		},
		{
			name: "single directory without slashes",
			give: "test",
			want: "/test",
		},
		{
			name: "multiple nested directories",
			give: "/a/b/c/d",
			want: "/a/b/c/d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := path.Format(tt.give)
			if got != tt.want {
				t.Errorf("Format(%q) = %q, want %q", tt.give, got, tt.want)
			}
		})
	}
}
