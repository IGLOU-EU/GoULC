/*
 * Copyright 2025 Adrien Kara
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package oauth2_test

import (
	"net/http"
	"testing"

	"gitlab.com/iglou.eu/goulc/duration"
	"gitlab.com/iglou.eu/goulc/hided"
	"gitlab.com/iglou.eu/goulc/http/client/auth/oauth2"
)

func TestResponse_Name(t *testing.T) {
	r := oauth2.Response{}
	if got := r.Name(); got != oauth2.ResponseName {
		t.Errorf("Response.Name() = %v, want %v", got, oauth2.ResponseName)
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
					Token:        hided.String("secret-token"),
					TokenType:    "Bearer",
					ExpiresIn:    duration.Duration{Duration: 3600},
					RefreshToken: hided.String("refresh-secret"),
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

			if !tt.wantErr {
				// Check token response fields
				if r.Token != tt.want.Token {
					t.Errorf("Token = %v, want %v", r.Token, tt.want.Token)
				}
				if r.TokenType != tt.want.TokenType {
					t.Errorf("TokenType = %v, want %v", r.TokenType, tt.want.TokenType)
				}
				if r.ExpiresIn != tt.want.ExpiresIn {
					t.Errorf("ExpiresIn = %v, want %v", r.ExpiresIn, tt.want.ExpiresIn)
				}
				if r.RefreshToken != tt.want.RefreshToken {
					t.Errorf("RefreshToken = %v, want %v", r.RefreshToken, tt.want.RefreshToken)
				}
				if r.Scope != tt.want.Scope {
					t.Errorf("Scope = %v, want %v", r.Scope, tt.want.Scope)
				}

				// Check error response fields
				if r.Error != tt.want.Error {
					t.Errorf("Error = %v, want %v", r.Error, tt.want.Error)
				}
				if r.ErrorDescription != tt.want.ErrorDescription {
					t.Errorf("ErrorDescription = %v, want %v", r.ErrorDescription, tt.want.ErrorDescription)
				}
				if r.ErrorURI != tt.want.ErrorURI {
					t.Errorf("ErrorURI = %v, want %v", r.ErrorURI, tt.want.ErrorURI)
				}
			}
		})
	}
}
