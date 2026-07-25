package oauth2_test

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"gitlab.com/iglou.eu/goulc/hided"
	"gitlab.com/iglou.eu/goulc/http/client"
	"gitlab.com/iglou.eu/goulc/http/client/auth/oauth2"
)

// newTokenServer starts a TLS test server acting as a token endpoint,
// counting hits so tests can assert on the cache behavior.
func newTokenServer(
	t *testing.T, statusCode int, response string, hits *atomic.Int32,
) *httptest.Server {
	t.Helper()

	server := httptest.NewTLSServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			hits.Add(1)

			if r.Method != http.MethodPost {
				t.Errorf("Expected POST request, got %s", r.Method)
			}
			if ct := r.Header.Get("Content-Type"); ct !=
				"application/x-www-form-urlencoded" {
				t.Errorf("Expected Content-Type"+
					" application/x-www-form-urlencoded, got %s", ct)
			}

			w.WriteHeader(statusCode)
			_, _ = w.Write([]byte(response))
		}))
	t.Cleanup(server.Close)

	return server
}

// newTestCredentials wires a ClientCredentials on the given test server,
// trusting its self-signed certificate.
func newTestCredentials(
	t *testing.T, server *httptest.Server,
) *oauth2.ClientCredentials {
	t.Helper()

	httpClient, err := client.New(
		context.Background(), server.URL, nil, &client.Options{
			DisableHTTPS:     true,
			DisableTLSVerify: true,
			Timeout:          time.Minute,
		}, nil)
	if err != nil {
		t.Fatalf("Failed to create HTTP client: %v", err)
	}
	t.Cleanup(func() { _ = httpClient.Close() })

	config := oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: hided.NewString("test-secret"),
		Endpoint: oauth2.Endpoint{
			URL:  server.URL,
			Auth: "/oauth/token",
		},
		Scopes: []string{"read", "write"},
	}

	cc, err := oauth2.NewClientCredentials(
		oauth2.ClientInHeader, config, slog.Default(), &httpClient)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	return cc
}

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
				ClientSecret: hided.NewString("test-secret"),
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
				ClientSecret: hided.NewString("test-secret"),
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
				ClientSecret: hided.NewString("test-secret"),
				Endpoint: oauth2.Endpoint{
					URL:  "https://example.com",
					Auth: "/oauth/token",
				},
			},
			logger:  nil,
			wantErr: false,
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
	tests := []struct {
		name           string
		mockResponse   string
		mockStatusCode int
		wantErr        error
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
		},
		{
			name:           "server error",
			mockResponse:   `{"error": "server_error", "error_description": "Internal error"}`,
			mockStatusCode: http.StatusInternalServerError,
			wantErr:        oauth2.ErrUnexpectedStatusCode,
		},
		{
			name:           "empty response",
			mockResponse:   "",
			mockStatusCode: http.StatusOK,
			wantErr:        oauth2.ErrEmptyBody,
		},
		{
			name:           "response without token",
			mockResponse:   `{"token_type": "Bearer", "expires_in": 3600}`,
			mockStatusCode: http.StatusOK,
			wantErr:        oauth2.ErrNoToken,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var hits atomic.Int32
			server := newTokenServer(
				t, tt.mockStatusCode, tt.mockResponse, &hits)
			cc := newTestCredentials(t, server)

			before := time.Now()
			err := cc.Update()
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Update() error = %v, want %v", err, tt.wantErr)
			}

			if tt.wantErr != nil {
				return
			}

			// The acquired token must be cached with its real lifetime,
			// expires_in is expressed in seconds (RFC 6749 §5.1)
			if got := cc.Token.Token.Reveal(); got != "new-token" {
				t.Errorf("Update() token = %q, want %q", got, "new-token")
			}

			wantExpire := before.Add(3600 * time.Second)
			if cc.Token.ExpireAt.Before(wantExpire.Add(-time.Minute)) ||
				cc.Token.ExpireAt.After(wantExpire.Add(time.Minute)) {
				t.Errorf("Update() ExpireAt = %v, want about %v",
					cc.Token.ExpireAt, wantExpire)
			}
		})
	}
}

func TestClientCredentials_TokenCache(t *testing.T) {
	var hits atomic.Int32
	server := newTokenServer(t, http.StatusOK, `{
		"access_token": "cached-token",
		"token_type": "Bearer",
		"expires_in": 3600
	}`, &hits)
	cc := newTestCredentials(t, server)

	// A valid cached token must be served without any new HTTP fetch
	for i := 0; i < 2; i++ {
		if err := cc.Update(); err != nil {
			t.Fatalf("Update() call %d unexpected error: %v", i+1, err)
		}

		_, value, err := cc.Header(http.MethodGet, nil, nil)
		if err != nil {
			t.Fatalf("Header() call %d unexpected error: %v", i+1, err)
		}
		if want := "Bearer cached-token"; value != want {
			t.Errorf("Header() call %d value = %q, want %q", i+1, value, want)
		}
	}

	if got := hits.Load(); got != 1 {
		t.Errorf("token endpoint hits = %d, want 1", got)
	}
}

func TestClientCredentials_UpdateClosedClient(t *testing.T) {
	httpClient, err := client.New(
		context.Background(), "https://example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("client.New() unexpected error: %v", err)
	}
	if err := httpClient.Close(); err != nil {
		t.Fatalf("Close() unexpected error: %v", err)
	}

	cc, err := oauth2.NewClientCredentials(oauth2.ClientInHeader, oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: hided.NewString("test-secret"),
		Endpoint: oauth2.Endpoint{
			URL:  "https://example.com",
			Auth: "/oauth/token",
		},
	}, nil, &httpClient)
	if err != nil {
		t.Fatalf("NewClientCredentials() unexpected error: %v", err)
	}

	// A refresh through a closed client must surface the closure
	// instead of panicking on the nil child
	if err := cc.Update(); !errors.Is(err, client.ErrClientClosed) {
		t.Errorf("Update() error = %v, want %v", err, client.ErrClientClosed)
	}

	// A clone of it carries no usable client and must report the same
	if err := cc.Clone().Update(); !errors.Is(err, client.ErrClientClosed) {
		t.Errorf("Clone().Update() error = %v, want %v",
			err, client.ErrClientClosed)
	}
}

func TestClientCredentials_ConcurrentUpdateHeader(t *testing.T) {
	var hits atomic.Int32
	// A zero lifetime forces a refresh on every Update, exercising
	// concurrent writes against concurrent Header reads
	server := newTokenServer(t, http.StatusOK, `{
		"access_token": "race-token",
		"token_type": "Bearer",
		"expires_in": 0
	}`, &hits)
	cc := newTestCredentials(t, server)

	var wg sync.WaitGroup
	for range 4 {
		wg.Add(2)

		go func() {
			defer wg.Done()
			for range 25 {
				if err := cc.Update(); err != nil {
					t.Errorf("Update() unexpected error: %v", err)
					return
				}
			}
		}()

		go func() {
			defer wg.Done()
			for range 25 {
				if _, _, err := cc.Header(
					http.MethodGet, nil, nil); err != nil {
					t.Errorf("Header() unexpected error: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
}

func TestClientCredentials_Header(t *testing.T) {
	logger := slog.Default()
	config := oauth2.Config{
		ClientID:     "test-client",
		ClientSecret: hided.NewString("test-secret"),
		Endpoint: oauth2.Endpoint{
			URL:  "https://example.com",
			Auth: "/oauth/token",
		},
	}

	cc, err := oauth2.NewClientCredentials(oauth2.ClientInHeader, config, logger, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Set up a mock token
	cc.Token = oauth2.TokenResponse{
		Token:     hided.NewString("test-token"),
		TokenType: "Bearer",
		ExpiresIn: 3600,
		ExpireAt:  time.Now().Add(time.Hour),
	}

	name, value, err := cc.Header(http.MethodGet, nil, nil)
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
		ClientSecret: hided.NewString("test-secret"),
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
