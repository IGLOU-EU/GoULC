package auth_test

import (
	"net/http"
	"testing"

	"gitlab.com/iglou.eu/goulc/hided"
	"gitlab.com/iglou.eu/goulc/http/client/auth"
)

func TestNewBasic(t *testing.T) {
	tests := []struct {
		name        string
		userID      string
		password    hided.String
		expectedErr error
	}{
		{
			name:        "valid credentials",
			userID:      "testuser",
			password:    hided.String("testpass"),
			expectedErr: nil,
		},
		{
			name:        "empty userID",
			userID:      "",
			password:    hided.String("testpass"),
			expectedErr: auth.ErrNoUserID,
		},
		{
			name:        "empty password",
			userID:      "testuser",
			password:    hided.String(""),
			expectedErr: auth.ErrNoPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := auth.NewBasic(tt.userID, tt.password)
			if err != tt.expectedErr {
				t.Errorf("NewBasic() expected error = %v, got = %v", tt.expectedErr, err)
				return
			}
			if tt.expectedErr == nil {
				if got.UserID != tt.userID {
					t.Errorf("NewBasic().UserID = %v, want %v", got.UserID, tt.userID)
				}
				if got.Password != tt.password {
					t.Errorf("NewBasic().Password = %v, want %v", got.Password, tt.password)
				}
			}
		})
	}
}

func TestBasic_Header(t *testing.T) {
	basic, _ := auth.NewBasic("testuser", hided.String("testpass"))

	name, value, err := basic.Header(http.MethodGet, nil, nil)
	if err != nil {
		t.Errorf("Basic.Header() unexpected error = %v", err)
		return
	}

	if name != auth.BasicHeaderName {
		t.Errorf("Basic.Header() name = %v, want %v", name, auth.BasicHeaderName)
	}

	expectedValue := auth.BasicValuePrefix + auth.BasicUserPass("testuser", "testpass")
	if value != expectedValue {
		t.Errorf("Basic.Header() value = %v, want %v", value, expectedValue)
	}
}

func TestBasic_Clone(t *testing.T) {
	original, _ := auth.NewBasic("testuser", hided.String("testpass"))
	cloned := original.Clone()

	// Check if the cloned instance is a different pointer
	if &original == cloned.(*auth.Basic) {
		t.Error("Clone() returned same pointer instead of new instance")
	}

	// Check if the values are the same
	if original.UserID != cloned.(*auth.Basic).UserID {
		t.Errorf("Clone() UserID = %v, want %v", cloned.(*auth.Basic).UserID, original.UserID)
	}
	if original.Password != cloned.(*auth.Basic).Password {
		t.Errorf("Clone() Password = %v, want %v", cloned.(*auth.Basic).Password, original.Password)
	}
}

func TestBasic_Name(t *testing.T) {
	basic, _ := auth.NewBasic("testuser", hided.String("testpass"))

	if got := basic.Name(); got != auth.BasicName {
		t.Errorf("basic.Name() = %v, want %v", got, auth.BasicName)
	}
}

// TestBasic_Update juste for coverage...
func TestBasic_Update(t *testing.T) {
	b := &auth.Basic{}
	_ = b.Update()
}

func TestBasicUserPass(t *testing.T) {
	tests := []struct {
		name     string
		userID   string
		password string
		want     string
	}{
		{
			name:     "standard credentials",
			userID:   "testuser",
			password: "testpass",
			want:     "dGVzdHVzZXI6dGVzdHBhc3M=", // base64("testuser:testpass")
		},
		{
			name:     "special characters",
			userID:   "test@user",
			password: "test:pass",
			want:     "dGVzdEB1c2VyOnRlc3Q6cGFzcw==", // base64("test@user:test:pass")
		},
		{
			name:     "empty strings",
			userID:   "",
			password: "",
			want:     "Og==", // base64(":")
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := auth.BasicUserPass(tt.userID, tt.password); got != tt.want {
				t.Errorf("BasicUserPass() = %v, want %v", got, tt.want)
			}
		})
	}
}
