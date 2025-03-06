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
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"gitlab.com/iglou.eu/goulc/duration"
	"gitlab.com/iglou.eu/goulc/hided"
	"gitlab.com/iglou.eu/goulc/http/client"
	"gitlab.com/iglou.eu/goulc/http/client/auth/oauth2"
	"gitlab.com/iglou.eu/goulc/http/methods"
)

func TestNewClientCredentials(t *testing.T) {
	logger := slog.Default()

	tests := []struct {
		name       string
		clientAuth oauth2.ClientCredentialsType
		config     oauth2.Config
		logger     *slog.Logger
		wantErr    bool
	}{
		{
			name:       "valid configuration with header auth",
			clientAuth: oauth2.ClientInHeader,
			config: oauth2.Config{
				ClientID:     "test-client",
				ClientSecret: hided.String("test-secret"),
				Endpoint: oauth2.Endpoint{
					URL:  "https://example.com",
					Auth: "/oauth/token",
				},
				Scopes: []string{"read", "write"},
			},
			logger:  logger,
			wantErr: false,
		},
		{
			name:       "valid configuration with body auth",
			clientAuth: oauth2.ClientInBody,
			config: oauth2.Config{
				ClientID:     "test-client",
				ClientSecret: hided.String("test-secret"),
				Endpoint: oauth2.Endpoint{
					URL:  "https://example.com",
					Auth: "/oauth/token",
				},
			},
			logger:  logger,
			wantErr: false,
		},
		{
			name:       "nil logger",
			clientAuth: oauth2.ClientInHeader,
			config: oauth2.Config{
				ClientID:     "test-client",
				ClientSecret: hided.String("test-secret"),
				Endpoint: oauth2.Endpoint{
					URL:  "https://example.com",
					Auth: "/oauth/token",
				},
			},
			logger:  nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := oauth2.NewClientCredentials(tt.clientAuth, tt.config, tt.logger, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewClientCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got == nil {
				t.Error("NewClientCredentials() returned nil but no error")
			}
		})
	}
}

func TestClientCredentials_Update(t *testing.T) {
	logger := slog.Default()

	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        bool
	}{
		{
			name: "successful token acquisition",
			mockResponse: `{
				"access_token": "new-token",
				"token_type": "Bearer",
				"expires_in": 3600,
				"scope": "read write"
			}`,
			mockStatusCode: http.StatusOK,
			wantErr:        false,
		},
		{
			name:           "server error",
			mockResponse:   `{"error": "server_error", "error_description": "Internal error"}`,
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        true,
		},
		{
			name:           "empty response",
			mockResponse:   "",
			mockStatusCode: http.StatusOK,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
					t.Errorf("Expected Content-Type application/x-www-form-urlencoded, got %s", r.Header.Get("Content-Type"))
				}

				w.WriteHeader(tt.mockStatusCode)
				w.Write([]byte(tt.mockResponse))
			}))
			defer server.Close()

			serverURL, _ := url.Parse(server.URL)
			// Create a custom HTTP client that trusts the test server's certificate
			httpClient, err := client.New(context.Background(), server.URL, nil, &client.Options{
				OnlyHTTPS:        false,
				DisableTLSVerify: true,
				Timeout:          time.Duration(1 * time.Minute),
				Context:          context.Background(),
			}, nil)
			if err != nil {
				t.Fatalf("Failed to create HTTP client: %v", err)
			}

			config := oauth2.Config{
				ClientID:     "test-client",
				ClientSecret: hided.String("test-secret"),
				Endpoint: oauth2.Endpoint{
					URL:  serverURL.String(),
					Auth: "/oauth/token",
				},
				Scopes: []string{"read", "write"},
			}

			client, err := oauth2.NewClientCredentials(oauth2.ClientInHeader, config, logger, &httpClient)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			err = client.Update()
			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClientCredentials_Header(t *testing.T) {
	logger := slog.Default()
	config := oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: hided.String("test-secret"),
		Endpoint: oauth2.Endpoint{
			URL:  "https://example.com",
			Auth: "/oauth/token",
		},
	}

	client, err := oauth2.NewClientCredentials(oauth2.ClientInHeader, config, logger, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Set up a mock token
	client.Token = oauth2.TokenResponse{
		Token:     hided.String("test-token"),
		TokenType: "Bearer",
		ExpiresIn: duration.Duration{Duration: 3600},
		ExpireAt:  time.Now().Add(time.Hour),
	}

	name, value, err := client.Header(methods.GET, nil, nil)
	if err != nil {
		t.Errorf("Header() unexpected error: %v", err)
	}
	if name != oauth2.ClientCredentialsHeaderName {
		t.Errorf("Header() name = %v, want %v", name, oauth2.ClientCredentialsHeaderName)
	}
	expectedPrefix := oauth2.ClientCredentialsHeaderPrefix
	if value[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Header() value prefix = %v, want %v", value[:len(expectedPrefix)], expectedPrefix)
	}
}

func TestClientCredentials_Clone(t *testing.T) {
	logger := slog.Default()
	config := oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: hided.String("test-secret"),
		Endpoint: oauth2.Endpoint{
			URL:  "https://example.com",
			Auth: "/oauth/token",
		},
		Scopes: []string{"read", "write"},
	}

	original, err := oauth2.NewClientCredentials(oauth2.ClientInHeader, config, logger, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	clone := original.Clone()
	if clone == nil {
		t.Fatal("Clone() returned nil")
	}

	// Verify the clone is a different instance
	if clone == original {
		t.Error("Clone() returned the same instance")
	}

	// Verify the clone has the same type
	if _, ok := clone.(*oauth2.ClientCredentials); !ok {
		t.Error("Clone() returned wrong type")
	}
}
