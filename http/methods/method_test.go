package methods_test

import (
	"testing"

	"gitlab.com/iglou.eu/goulc/http/methods"
)

func TestMethod_IsValid(t *testing.T) {
	tests := []struct {
		name   string
		method methods.Method
		want   bool
	}{
		{
			name:   "empty method",
			method: "",
			want:   false,
		},
		{
			name:   "GET method",
			method: methods.GET,
			want:   true,
		},
		{
			name:   "HEAD method",
			method: methods.HEAD,
			want:   true,
		},
		{
			name:   "POST method",
			method: methods.POST,
			want:   true,
		},
		{
			name:   "invalid method",
			method: "INVALID",
			want:   false,
		},
		{
			name:   "lowercase method",
			method: "get",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.method.IsValid(); got != tt.want {
				t.Errorf("Method.IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}
