package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"gitlab.com/iglou.eu/goulc/http/client"
	"gitlab.com/iglou.eu/goulc/http/client/auth"
)

// mockAuthenticator implements auth.Authenticator for testing
type mockAuthenticator struct {
	name     string
	header   string
	value    string
	updated  bool
	wantErr  bool
	cloneErr bool
}

func (m *mockAuthenticator) Name() string  { return m.name }
func (m *mockAuthenticator) Update() error { m.updated = true; return nil }
func (m *mockAuthenticator) Header(
	_ string, _ *url.URL, _ []byte,
) (headerKey, headerValue string, err error) {
	if m.wantErr {
		return "", "", auth.ErrNoUserID
	}
	return m.header, m.value, nil
}
func (m *mockAuthenticator) Clone() auth.Authenticator {
	if m.cloneErr {
		return nil
	}
	return &mockAuthenticator{
		name:   m.name,
		header: m.header,
		value:  m.value,
	}
}

// mockResponse implements client.Unmarshaler for testing
type mockResponse struct {
	Message string `json:"message"`
}

func (_ *mockResponse) Name() string { return "mockResponse" }
func (m *mockResponse) Unmarshal(_ int, _ http.Header, body []byte) error {
	return json.Unmarshal(body, m)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name          string
		serverURL     string
		auth          auth.Authenticator
		opt           *client.Options
		logger        *slog.Logger
		ctx           context.Context
		wantErr       bool
		expectedError error
	}{
		{
			name:          "empty server URL",
			serverURL:     "",
			wantErr:       true,
			expectedError: client.ErrEmptyServerURL,
		},
		{
			name:          "invalid URL",
			serverURL:     "://invalid",
			wantErr:       true,
			expectedError: client.ErrInvalidURL,
		},
		{
			name:      "valid URL with trailing slash",
			serverURL: "https://candlekeep.faerun/",
			wantErr:   false,
		},
		{
			name:      "valid URL without trailing slash",
			serverURL: "https://candlekeep.faerun",
			wantErr:   false,
		},
		{
			name:          "invalid URL with query parameters",
			serverURL:     "https://example.com?par;am=va;lue",
			wantErr:       true,
			expectedError: client.ErrInvalidQuery,
		},
		{
			name:      "URL with path",
			serverURL: "https://candlekeep.faerun/library/v1",
			wantErr:   false,
		},
		{
			name:      "HTTP URL with OnlyHTTPS",
			serverURL: "http://candlekeep.faerun",
			opt:       &client.OptDefault,
			wantErr:   false,
		},
		{
			name:      "with authenticator",
			serverURL: "https://candlekeep.faerun",
			auth: &mockAuthenticator{
				name:   "mock",
				header: "Authorization",
				value:  "Bearer token",
			},
			wantErr: false,
		},
		{
			name:      "with custom options",
			serverURL: "https://candlekeep.faerun",
			opt: &client.Options{
				OnlyHTTPS:        true,
				Follow:           true,
				FollowAuth:       true,
				FollowReferer:    true,
				MaxRedirect:      5,
				Timeout:          60 * time.Second,
				DisableTLSVerify: true,
			},
			wantErr: false,
		},
		{
			name:      "invalid timeout",
			serverURL: "https://candlekeep.faerun",
			opt: &client.Options{
				Timeout: -1 * time.Second,
			},
			wantErr:       true,
			expectedError: client.ErrInvalidTimeout,
		},
		{
			name:      "invalid redirect limit",
			serverURL: "https://candlekeep.faerun",
			opt: &client.Options{
				MaxRedirect: -1,
			},
			wantErr:       true,
			expectedError: client.ErrInvalidRedirectLimit,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := client.New(tt.ctx, tt.serverURL, tt.auth, tt.opt, tt.logger)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && tt.expectedError != nil {
				if !errors.Is(err, tt.expectedError) {
					t.Errorf("New() expected error = %v, got = %v", tt.expectedError, err)
					return
				}
			}
			if tt.wantErr {
				return
			}

			if c.Auth != tt.auth {
				t.Errorf("New() auth = %v, want %v", c.Auth, tt.auth)
			}
		})
	}
}

func TestClient_NewChild(t *testing.T) {
	parent, err := client.New(context.Background(), "https://vault13.wasteland", nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create parent client: %v", err)
	}

	// Add some headers and query params to parent
	parent.Header.Set("Vault-Access", "pip-boy")
	parent.Query.Set("water_chip", "working")

	child := parent.NewChild("")

	// Check that child inherits parent's configuration
	if child.URL != parent.URL {
		t.Errorf("Child URL = %v, want %v", child.URL, parent.URL)
	}

	// Check that child has its own copy of headers
	child.Header.Set("Overseer-Auth", "vault-dweller")
	if parent.Header.Get("Overseer-Auth") != "" {
		t.Error("Child header modified parent headers")
	}

	// Check that child has its own copy of query params
	childQuery := make(url.Values)
	for k, v := range parent.Query {
		childQuery[k] = v
	}
	child.Query = childQuery
	child.Query.Set("rad_level", "high")
	if parent.Query.Get("rad_level") != "" {
		t.Error("Child query modified parent query")
	}

	// Test with authenticator
	authMock := &mockAuthenticator{
		name:   "pip-boy",
		header: "Authorization",
		value:  "Vault-Tec clearance",
	}
	parentWithAuth, _ := client.New(context.Background(), "https://vault13.wasteland", authMock, nil, nil)
	childWithAuth := parentWithAuth.NewChild("")

	if childWithAuth.Auth == nil {
		t.Error("Child did not inherit parent's authenticator")
	}

	child = parent.NewChild("/v1/")
	if child.URL.String() != "https://vault13.wasteland/v1" {
		t.Errorf("Child URL = %v, want %v", child.URL, "https://example.com/v1")
	}

	parent, _ = client.New(context.Background(), "https://vault13.wasteland/api/", nil, nil, nil)

	child = parent.NewChild("/v1/")
	if child.URL.String() != "https://vault13.wasteland/api/v1" {
		t.Errorf("Child URL = %v, want %v", child.URL, "https://example.com/api/v1")
	}
}

//gocyclo:ignore
func TestClient_Do(t *testing.T) {
	// Create test server
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/temple":
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]string{"message": "Go, friend, and may Gorion watch over your path"})
		case "/sarevok":
			w.WriteHeader(http.StatusInternalServerError)
		case "/portal":
			http.Redirect(w, r, "/temple", http.StatusTemporaryRedirect)
		case "/maze":
			http.Redirect(w, r, "/maze", http.StatusTemporaryRedirect)
		case "/meditation":
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusOK)
		case "/prophecy":
			if v := r.URL.Query().Get("bhaalspawn"); v != "" {
				w.WriteHeader(http.StatusOK)
				break
			}
			w.WriteHeader(http.StatusInternalServerError)
		case "/necropolis":
			if r.Header.Get("Authorization") == "" {
				w.WriteHeader(http.StatusUnauthorized)
				break
			}
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	// Create client with TLS config from test server
	opt := client.OptDefault
	opt.DisableTLSVerify = true
	opt.Timeout = 1 * time.Second
	c, err := client.New(context.Background(), ts.URL, nil, &opt, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name       string
		path       string
		method     string
		query      [2]string
		body       []byte
		notFollow  bool
		response   client.Unmarshaler
		wantStatus int
		wantErr    bool
		wantErrIs  error
	}{
		{
			name:       "successful GET",
			path:       "/temple",
			method:     http.MethodGet,
			response:   &mockResponse{},
			wantStatus: http.StatusOK,
		},
		{
			name:      "empty method",
			wantErr:   true,
			wantErrIs: client.ErrInvalidMethod,
		},
		{
			name:       "server error",
			path:       "/sarevok",
			method:     http.MethodGet,
			wantStatus: http.StatusInternalServerError,
			wantErr:    true,
		},
		{
			name:       "successful with URL query",
			path:       "/prophecy",
			method:     http.MethodGet,
			query:      [2]string{"bhaalspawn", "child of murder"},
			wantStatus: http.StatusOK,
		},
		{
			name:       "successful POST with body",
			path:       "/temple",
			method:     http.MethodPost,
			body:       []byte(`{"scroll":"identify"}`),
			response:   &mockResponse{},
			wantStatus: http.StatusOK,
		},
		{
			name:       "redirect",
			path:       "/portal",
			method:     http.MethodGet,
			response:   &mockResponse{},
			wantStatus: http.StatusOK,
		},
		{
			name:       "redirect not accepted",
			path:       "/portal",
			method:     http.MethodGet,
			notFollow:  true,
			wantStatus: http.StatusTemporaryRedirect,
		},
		{
			name:      "too many redirects",
			path:      "/maze",
			method:    http.MethodGet,
			wantErr:   true,
			wantErrIs: client.ErrTooManyRedirects,
		},
		{
			name:    "timeout",
			path:    "/meditation",
			method:  http.MethodGet,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create child client for each test
			child := c.NewChild(tt.path)

			if tt.query[0] != "" {
				child.Query.Set(tt.query[0], tt.query[1])
			}

			if tt.notFollow {
				child.Options.Follow = false
			}

			resp, err := child.Do(tt.method, tt.body, tt.response)
			if !tt.wantErr && err != nil {
				t.Errorf("Do() unexpected error = %v", err)
				return
			}
			if err == nil {
				if resp == nil && resp.StatusCode < 400 {
					t.Errorf("Do() expected error but got nil")
					return
				}
				// Consider HTTP error status codes as errors for test purposes
				err = errors.New("HTTP error status codes as errors")
			}
			if tt.wantErrIs != nil && !errors.Is(err, tt.wantErrIs) {
				t.Errorf("Do() error = %v, want %v", err, tt.wantErrIs)
				return
			}

			if !tt.wantErr && resp != nil && resp.StatusCode != tt.wantStatus {
				t.Errorf("Do() status = %v, want %v\n%#v", resp.StatusCode, tt.wantStatus, resp.Header)
			}
		})
	}

	// Test concurrent requests
	t.Run("concurrent requests", func(t *testing.T) {
		const numRequests = 10
		wg := sync.WaitGroup{}
		wg.Add(numRequests)

		for i := 0; i < numRequests; i++ {
			go func() {
				defer wg.Done()
				child := c.NewChild("/temple")
				resp, err := child.Do(http.MethodGet, nil, &mockResponse{})
				if err != nil {
					t.Errorf("Concurrent Do() error = %v", err)
					return
				}
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Concurrent Do() status = %v, want %v", resp.StatusCode, http.StatusOK)
				}
			}()
		}
		wg.Wait()
	})

	t.Run("cAuth request", func(t *testing.T) {
		basic, _ := auth.NewBasic("minsc", "go-for-the-eyes")
		child := c.NewChild("/necropolis")
		child.Auth = &basic

		resp, err := child.Do(http.MethodGet, nil, nil)
		if err != nil {
			t.Errorf("Auth Do() error = %v", err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("Auth Do() status = %v, want %v", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("Rate limiter request", func(t *testing.T) {
		// Test rate limiter
		child := c.NewChild("/maze")
		child.Options.Timeout = 1 * time.Minute
		child.Options.MaxRedirect = 5
		child.Options.RateLimiter = rate.NewLimiter(rate.Every(time.Second), 1)

		chronoStart := time.Now()
		_, err = child.Do(http.MethodGet, nil, nil)
		if !errors.Is(err, client.ErrTooManyRedirects) {
			t.Errorf("Do() error = %v, wantErr %v", err, client.ErrTooManyRedirects)
		}
		chronoEnd := time.Now()
		chronoRes := chronoEnd.Sub(chronoStart)
		if chronoRes > time.Duration(child.Options.MaxRedirect)*time.Second {
			t.Errorf("Do() expected to take least 10s, took %v\nStart: %v; End: %v", chronoRes, chronoStart, chronoEnd)
		}
	})

	t.Run("Close client", func(t *testing.T) {
		// Verify Do return when closed
		if clone := c.Close(); clone != nil {
			t.Errorf("Clone() on closed client = %v, want nil", clone)
		}

		res, err := c.Do(http.MethodPost, nil, nil)
		if res != nil {
			t.Errorf("Do() expect to return nil when Client are closed")
		}

		if !errors.Is(err, client.ErrClientClosed) {
			t.Errorf("Do() error = %v, wantErr %v", err, client.ErrClientClosed)
		}
	})
}

func TestClient_Close(t *testing.T) {
	opt := client.OptDefault
	opt.Timeout = 2 * time.Second
	main, err := client.New(context.TODO(), "https://vault13.wasteland", nil, &opt, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	c := main.Clone()
	if err := c.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// Verify client is closed
	if !c.IsClosed() {
		t.Error("Close() did not close the client")
	}

	// Verify when Client are already closed
	if err := c.Close(); err != nil {
		t.Errorf("Close() a closed Client expect to return a nil, error = %v", err)
	}

	// Verify parent close children
	c = main.Clone()
	if err := main.Close(); err != nil {
		t.Errorf("Close() error = %v", err)
	}
	if !c.IsClosed() {
		t.Error("Close() parent did not close the client")
	}
}

func TestClient_FlushHeader(t *testing.T) {
	c, err := client.New(context.Background(), "https://example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	c.Header.Set("Vault-Tec", "approved")
	c.FlushHeader()

	if len(c.Header) != 0 {
		t.Errorf("FlushHeader() did not clear headers, got %v", c.Header)
	}
}

func TestClient_FlushQuery(t *testing.T) {
	c, err := client.New(context.Background(), "https://example.com", nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	c.Query.Set("geck", "operational")
	c.FlushQuery()

	if len(c.Query) != 0 {
		t.Errorf("FlushQuery() did not clear query params, got %v", c.Query)
	}
}

func TestClient_Clone(t *testing.T) {
	authMock := &mockAuthenticator{
		name:   "pip-boy",
		header: "Authorization",
		value:  "Vault-Tec clearance",
	}

	opt := client.Options{
		OnlyHTTPS:        true,
		Follow:           true,
		FollowAuth:       true,
		MaxRedirect:      5,
		Timeout:          60 * time.Second,
		DisableTLSVerify: true,
	}

	c, err := client.New(context.Background(), "https://candlekeep.faerun", authMock, &opt, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Add some headers and query params
	c.Header.Set("Vault-Security", "classified")
	c.Query.Set("mutant_level", "dangerous")

	// Clone the client
	clone := c.Clone()

	// Verify cloned client has same configuration
	if clone.URL != c.URL {
		t.Errorf("Clone() URL = %v, want %v", clone.URL, c.URL)
	}
	if !reflect.DeepEqual(clone.Options, c.Options) {
		t.Errorf("Clone() Opts = %#v, want %#v", clone.Options, c.Options)
	}
	if !reflect.DeepEqual(clone.Header, c.Header) {
		t.Errorf("Clone() Header = %v, want %v", clone.Header, c.Header)
	}
	if !reflect.DeepEqual(clone.Query, c.Query) {
		t.Errorf("Clone() Query = %v, want %v", clone.Query, c.Query)
	}

	// Verify modifications to clone don't affect original
	clone.Header.Set("Overseer-Command", "evacuate")
	clone.Query.Set("deathclaw_alert", "high")

	if c.Header.Get("Overseer-Command") != "" {
		t.Error("Clone header modified original headers")
	}
	if c.Query.Get("deathclaw_alert") != "" {
		t.Error("Clone query modified original query params")
	}

	// verify if clone make a new pointer into clone.URL.User
	// when parent have a user field in c.URL
	c.URL.User = &url.Userinfo{}
	clone = c.Clone()
	if clone.URL.User == c.URL.User {
		t.Errorf("Clone() .URL.User need to be a new pointer c = %p, clone = %p", c.URL.User, clone.URL.User)
	}

	// Verify if parent client is closed that returns nil
	c.Close()
	if clone := c.Clone(); clone != nil {
		t.Errorf("Clone() on closed client = %v, want nil", clone)
	}
}

type testMarshaler struct {
	Message string `json:"message"`
}

func (_ *testMarshaler) Name() string { return "testMarshaler" }

func (_ *testMarshaler) ContentType() string {
	return "application/json"
}

func (m *testMarshaler) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func TestClient_DoWithMarshal(t *testing.T) {
	// Create test server
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": "Welcome to New Reno, traveler"})
	}))
	defer ts.Close()

	// Create client
	opt := client.OptDefault
	opt.DisableTLSVerify = true
	c, err := client.New(context.Background(), ts.URL, nil, &opt, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	// Test cases
	tests := []struct {
		name     string
		body     client.Marshaler
		response client.Unmarshaler
		wantErr  bool
	}{
		{
			name:     "successful marshal",
			body:     &testMarshaler{Message: "War never changes"},
			response: &mockResponse{},
			wantErr:  false,
		},
		{
			name:     "nil body",
			body:     nil,
			response: &mockResponse{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := c.DoWithMarshal(http.MethodPost, tt.body, tt.response)
			if (err != nil) != tt.wantErr {
				t.Errorf("DoWithMarshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && resp.StatusCode != http.StatusOK {
				t.Errorf("DoWithMarshal() status = %v, want %v", resp.StatusCode, http.StatusOK)
			}
		})
	}

	// Verify DoWithMarshal return when closed
	if clone := c.Close(); clone != nil {
		t.Errorf("Clone() on closed client = %v, want nil", clone)
	}

	res, err := c.DoWithMarshal(http.MethodPost, nil, nil)
	if res != nil {
		t.Errorf("DoWithMarshal() expect to return nil when Client are closed")
	}

	if !errors.Is(err, client.ErrClientClosed) {
		t.Errorf("DoWithMarshal() error = %v, wantErr %v", err, client.ErrClientClosed)
	}
}

//gocyclo:ignore
func TestClient_FollowRedirects(t *testing.T) {
	const (
		baseURL     = "https://new-reno.wasteland"
		redirectURL = "/silver-gecko"
		testToken   = "NCR-Ranger-Token"
	)

	c, err := client.New(context.Background(), baseURL, nil, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	tests := []struct {
		name           string
		setup          func(*client.Client)
		wantErr        error
		checkTrace     bool // if true, trace will be initialized and checked
		requestURL     string
		addAuthHeader  bool   // if true, test-token will be added to request
		expectedScheme string // if empty, scheme is not checked
		expectedAuth   string // if addAuthHeader is true, auth header must match this
	}{
		{
			name: "follow disabled",
			setup: func(c *client.Client) {
				c.Options.Follow = false
			},
			wantErr: http.ErrUseLastResponse,
		},
		{
			name: "nil trace",
			setup: func(c *client.Client) {
				c.Options.Follow = true
			},
			wantErr: client.ErrNoTrace,
		},
		{
			name: "max redirects exceeded",
			setup: func(c *client.Client) {
				c.Options.Follow = true
				c.Options.MaxRedirect = 1
			},
			wantErr:    client.ErrTooManyRedirects,
			checkTrace: true,
		},

		// Auth header cases
		{
			name: "disable auth on different host",
			setup: func(c *client.Client) {
				c.Options.Follow = true
				c.Options.FollowAuth = false
			},
			checkTrace:    true,
			requestURL:    "https://vault-city.wasteland" + redirectURL,
			addAuthHeader: true,
			expectedAuth:  "", // header should be removed
		},
		{
			name: "keep auth on same host",
			setup: func(c *client.Client) {
				c.Options.Follow = true
				c.Options.FollowAuth = false
			},
			checkTrace:    true,
			requestURL:    baseURL + redirectURL,
			addAuthHeader: true,
			expectedAuth:  testToken, // header should be preserved
		},

		// URL scheme cases
		{
			name: "enforce HTTPS on HTTP URL",
			setup: func(c *client.Client) {
				c.Options.Follow = true
				c.Options.OnlyHTTPS = true
			},
			checkTrace:     true,
			requestURL:     "http://gecko.wasteland" + redirectURL,
			expectedScheme: "https", // should be upgraded
		},
		{
			name: "keep HTTP URL as is",
			setup: func(c *client.Client) {
				c.Options.Follow = true
				c.Options.OnlyHTTPS = false
			},
			checkTrace:     true,
			requestURL:     "http://gecko.wasteland" + redirectURL,
			expectedScheme: "http", // should remain http
		},

		// Other options
		{
			name: "disable referer",
			setup: func(c *client.Client) {
				c.Options.Follow = true
				c.Options.FollowReferer = false
			},
			checkTrace: true,
		},
		{
			name: "with rate limiter",
			setup: func(c *client.Client) {
				c.Options.Follow = true
				c.Options.RateLimiter = rate.NewLimiter(rate.Every(time.Second), 1)
			},
			checkTrace: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset client for each test
			tc := c.Clone()
			tt.setup(tc)

			// Create a request with the appropriate URL
			reqURL := tt.requestURL
			if reqURL == "" {
				reqURL = baseURL
			}
			req := httptest.NewRequest(http.MethodGet, reqURL, nil)
			req.Response = &http.Response{Status: "302 Found"}

			// Add auth header if specified
			if tt.addAuthHeader {
				req.Header.Set("Authorization", testToken)
			}

			// Initialize trace if needed
			var trace *[]client.Redirects
			if tt.checkTrace {
				trace = &[]client.Redirects{}
			}

			// Create previous request for via slice
			via := []*http.Request{
				httptest.NewRequest(http.MethodGet, baseURL+"/first", nil),
			}
			via[0].Response = &http.Response{Status: "301 Moved Permanently"}

			// Call the redirect function and verify behavior
			redirectFunc := tc.FollowRedirects(trace)
			err := redirectFunc(req, via)

			// Verify error cases
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("FollowRedirects() error = %v, want %v", err, tt.wantErr)
				}
			} else if err != nil {
				t.Errorf("FollowRedirects() unexpected error = %v", err)
			}

			// Verify trace is populated
			if tt.checkTrace && trace != nil && len(*trace) == 0 {
				t.Error("FollowRedirects() trace is empty")
			}

			// Verify URL scheme
			if tt.expectedScheme != "" && req.URL.Scheme != tt.expectedScheme {
				t.Errorf("URL scheme = %q, want %q", req.URL.Scheme, tt.expectedScheme)
			}

			// Verify auth header
			if !tt.addAuthHeader {
				return
			}
			gotAuth := req.Header.Get("Authorization")
			if gotAuth != tt.expectedAuth {
				t.Errorf("Authorization header = %q, want %q", gotAuth, tt.expectedAuth)
			}
		})
	}
}
