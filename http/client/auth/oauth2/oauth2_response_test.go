package oauth2_test

import (
	"net/http"
	"testing"

	"gitlab.com/iglou.eu/goulc/hided"
	"gitlab.com/iglou.eu/goulc/http/client/auth/oauth2"
)

func TestResponse_Name(t *testing.T) {
	r := oauth2.Response{}
	if got := r.Name(); got != oauth2.ResponseName {
		t.Errorf("Response.Name() = %v, want %v", got, oauth2.ResponseName)
	}
}

func TestErrorResponse_IsEmpty(t *testing.T) {
	tests := []struct {
		name string
		give oauth2.ErrorResponse
		want bool
	}{
		{
			name: "zero value reports empty",
			give: oauth2.ErrorResponse{},
			want: true,
		},
		{
			name: "error code reports not empty",
			give: oauth2.ErrorResponse{Error: "invalid_request"},
			want: false,
		},
		{
			name: "description without the required error code reports empty",
			give: oauth2.ErrorResponse{ErrorDescription: "broken server"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.give.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestResponse_Unmarshal(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    oauth2.Response
		wantErr bool
	}{
		{
			name: "successful token response",
			json: `{
				"access_token": "secret-token",
				"token_type": "Bearer",
				"expires_in": 3600,
				"refresh_token": "refresh-secret",
				"scope": "read write"
			}`,
			want: oauth2.Response{
				TokenResponse: oauth2.TokenResponse{
					Token:        hided.NewString("secret-token"),
					TokenType:    "Bearer",
					ExpiresIn:    3600,
					RefreshToken: hided.NewString("refresh-secret"),
					Scope:        "read write",
				},
			},
			wantErr: false,
		},
		{
			name: "error response",
			json: `{
				"error": "invalid_request",
				"error_description": "Request was malformed",
				"error_uri": "https://example.com/errors/invalid_request"
			}`,
			want: oauth2.Response{
				ErrorResponse: oauth2.ErrorResponse{
					Error:            "invalid_request",
					ErrorDescription: "Request was malformed",
					ErrorURI:         "https://example.com/errors/invalid_request",
				},
			},
			wantErr: false,
		},
		{
			name:    "invalid json",
			json:    `{invalid json}`,
			want:    oauth2.Response{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r oauth2.Response
			err := r.Unmarshal(http.StatusOK, nil, []byte(tt.json))

			if (err != nil) != tt.wantErr {
				t.Errorf("Response.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Check token response fields
			got, want := r.TokenResponse, tt.want.TokenResponse
			if got.Token.Reveal() != want.Token.Reveal() {
				t.Errorf("Token = %v, want %v", got.Token, want.Token)
			}
			if got.TokenType != want.TokenType {
				t.Errorf("TokenType = %v, want %v", got.TokenType, want.TokenType)
			}
			if got.ExpiresIn != want.ExpiresIn {
				t.Errorf("ExpiresIn = %v, want %v", got.ExpiresIn, want.ExpiresIn)
			}
			if got.RefreshToken.Reveal() != want.RefreshToken.Reveal() {
				t.Errorf("RefreshToken = %v, want %v",
					got.RefreshToken, want.RefreshToken)
			}
			if got.Scope != want.Scope {
				t.Errorf("Scope = %v, want %v", got.Scope, want.Scope)
			}

			// Check error response fields
			if r.ErrorResponse != tt.want.ErrorResponse {
				t.Errorf("ErrorResponse = %+v, want %+v",
					r.ErrorResponse, tt.want.ErrorResponse)
			}
		})
	}
}
