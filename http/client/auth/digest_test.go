package auth_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"gitlab.com/iglou.eu/goulc/http/client/auth"
)

func TestNewDigest(t *testing.T) {
	tests := []struct {
		name       string
		username   string
		password   string
		parameters auth.DigestParameters
		wantErr    error
	}{
		{
			name:     "Valid digest",
			username: "Mufasa",
			password: "Circle of Life",
			parameters: auth.DigestParameters{
				Algorithm: auth.DigestMD5,
				Realm:     "testrealm@host.com",
				URI:       "/dir/index.html",
				Nonce:     "dcd98b7102dd2f0e8b11d0f600bfb0c093",
			},
			wantErr: nil,
		},
		{
			name:     "Empty username",
			username: "",
			password: "Circle of Life",
			parameters: auth.DigestParameters{
				Algorithm: auth.DigestMD5,
				Realm:     "testrealm@host.com",
				URI:       "/dir/index.html",
				Nonce:     "dcd98b7102dd2f0e8b11d0f600bfb0c093",
			},
			wantErr: auth.ErrNoUserID,
		},
		{
			name:     "Empty password",
			username: "Mufasa",
			password: "",
			parameters: auth.DigestParameters{
				Algorithm: auth.DigestMD5,
				Realm:     "testrealm@host.com",
				URI:       "/dir/index.html",
				Nonce:     "dcd98b7102dd2f0e8b11d0f600bfb0c093",
			},
			wantErr: auth.ErrNoPassword,
		},
		{
			name:     "Invalid algorithm",
			username: "Mufasa",
			password: "Circle of Life",
			parameters: auth.DigestParameters{
				Algorithm: "invalid",
				Realm:     "testrealm@host.com",
				URI:       "/dir/index.html",
				Nonce:     "dcd98b7102dd2f0e8b11d0f600bfb0c093",
			},
			wantErr: auth.ErrUnknownAlgorithm,
		},
		{
			name:     "Empty realm",
			username: "Mufasa",
			password: "Circle of Life",
			parameters: auth.DigestParameters{
				Algorithm: auth.DigestMD5,
				Realm:     "",
				URI:       "/dir/index.html",
				Nonce:     "dcd98b7102dd2f0e8b11d0f600bfb0c093",
			},
			wantErr: auth.ErrNoRealm,
		},
		{
			name:     "Empty nonce",
			username: "Mufasa",
			password: "Circle of Life",
			parameters: auth.DigestParameters{
				Algorithm: auth.DigestMD5,
				Realm:     "testrealm@host.com",
				URI:       "/dir/index.html",
				Nonce:     "",
			},
			wantErr: auth.ErrNoNonce,
		},
		{
			name:     "Empty URI",
			username: "Mufasa",
			password: "Circle of Life",
			parameters: auth.DigestParameters{
				Algorithm: auth.DigestMD5,
				Realm:     "testrealm@host.com",
				URI:       "",
				Nonce:     "dcd98b7102dd2f0e8b11d0f600bfb0c093",
			},
			wantErr: auth.ErrNoURI,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := auth.NewDigest(tt.username, tt.password, tt.parameters)
			if err != tt.wantErr {
				t.Errorf("NewDigest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// If we expect an error, no need to check the returned digest
			if tt.wantErr != nil {
				return
			}

			// Verify all fields are properly set for valid cases
			if got.Username != tt.username {
				t.Errorf("NewDigest().Username = %v, want %v", got.Username, tt.username)
			}
			if got.Password != tt.password {
				t.Errorf("NewDigest().Password = %v, want %v", got.Password, tt.password)
			}
			if got.Parameters.Algorithm != tt.parameters.Algorithm {
				t.Errorf("NewDigest().Parameters.Algorithm = %v, want %v", got.Parameters.Algorithm, tt.parameters.Algorithm)
			}
			if got.Parameters.Realm != tt.parameters.Realm {
				t.Errorf("NewDigest().Parameters.Realm = %v, want %v", got.Parameters.Realm, tt.parameters.Realm)
			}
			if got.Parameters.URI != tt.parameters.URI {
				t.Errorf("NewDigest().Parameters.URI = %v, want %v", got.Parameters.URI, tt.parameters.URI)
			}
			if got.Parameters.Nonce != tt.parameters.Nonce {
				t.Errorf("NewDigest().Parameters.Nonce = %v, want %v", got.Parameters.Nonce, tt.parameters.Nonce)
			}
		})
	}
}

func TestDigest_Name(t *testing.T) {
	d := &auth.Digest{}

	if got := d.Name(); got != auth.DigestName {
		t.Errorf("digest.Name() = %v, want %v", got, auth.DigestName)
	}
}

// TestBasic_Update juste for coverage...
func TestDigest_Update(t *testing.T) {
	d := &auth.Digest{}
	_ = d.Update()
}

func TestDigest_Clone(t *testing.T) {
	original := &auth.Digest{
		Username: "testuser",
		Password: "testpass",
		Parameters: auth.DigestParameters{
			Algorithm: auth.DigestSHA256,
			Realm:     "testrealm",
			URI:       "/test",
			QOP:       "auth",
			Nonce:     "testnonce",
			CNonce:    "testcnonce",
			NC:        "00000001",
			UserHash:  true,
			Opaque:    "testopaque",
		},
	}

	cloned := original.Clone()

	// Check if the cloned instance is a different pointer
	if original == cloned.(*auth.Digest) {
		t.Error("Clone() returned same pointer instead of new instance")
	}

	// Check if all values are the same
	d := cloned.(*auth.Digest)
	if &d == &original {
		t.Errorf("Clone() returned same pointer instead of new instance\nOriginal: %v\nCloned: %v", &original, &d)
	}
	if *d != *original {
		t.Errorf("Clone() didn't return a complete copy of the original instance\nOriginal: %v\nCloned: %v", *original, *d)
	}
}

func TestDigestParameters_Hash(t *testing.T) {
	testData := []byte("test data")
	tests := []struct {
		name      string
		algorithm auth.DigestAlgo
		wantLen   int // Expected length of the hash in characters (hex encoded)
	}{
		{
			name:      "MD5",
			algorithm: auth.DigestMD5,
			wantLen:   32, // MD5 hash is 16 bytes = 32 hex chars
		},
		{
			name:      "MD5-SESS",
			algorithm: auth.DigestMD5_SESS,
			wantLen:   32,
		},
		{
			name:      "SHA-256",
			algorithm: auth.DigestSHA256,
			wantLen:   64, // SHA-256 hash is 32 bytes = 64 hex chars
		},
		{
			name:      "SHA-256-SESS",
			algorithm: auth.DigestSHA256_SESS,
			wantLen:   64,
		},
		{
			name:      "SHA-512",
			algorithm: auth.DigestSHA512,
			wantLen:   128, // SHA-512 hash is 64 bytes = 128 hex chars
		},
		{
			name:      "SHA-512-SESS",
			algorithm: auth.DigestSHA512_SESS,
			wantLen:   128,
		},
		{
			name:      "SHA-512-256",
			algorithm: auth.DigestSHA512256,
			wantLen:   64, // SHA-512/256 hash is 32 bytes = 64 hex chars
		},
		{
			name:      "SHA-512-256-SESS",
			algorithm: auth.DigestSHA512256_SESS,
			wantLen:   64,
		},
		{
			name:      "Unknown algorithm",
			algorithm: "invalid",
			wantLen:   len(auth.ErrUnknownAlgorithm.Error()),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &auth.DigestParameters{Algorithm: tt.algorithm}
			got := d.Hash(testData)
			if len(got) != tt.wantLen {
				t.Errorf("Hash() length = %v, want %v", len(got), tt.wantLen)
			}
		})
	}
}

func TestDigest_A1(t *testing.T) {
	tests := []struct {
		name     string
		digest   *auth.Digest
		wantHash bool
	}{
		{
			name: "Basic A1",
			digest: &auth.Digest{
				Username: "Mufasa",
				Password: "Circle of Life",
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestMD5,
					Realm:     "testrealm@host.com",
				},
			},
			wantHash: true,
		},
		{
			name: "Session A1",
			digest: &auth.Digest{
				Username: "Mufasa",
				Password: "Circle of Life",
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestMD5_SESS,
					Realm:     "testrealm@host.com",
					Nonce:     "nonce",
					CNonce:    "cnonce",
				},
			},
			wantHash: true,
		},
		{
			name: "UserHash A1",
			digest: &auth.Digest{
				Username: "Müfasa",
				Password: "Circle of Life",
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestSHA256,
					Realm:     "testrealm@host.com",
					UserHash:  true,
				},
			},
			wantHash: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.digest.A1()
			if (got != "") != tt.wantHash {
				t.Errorf("A1() = %v, want hash: %v", got, tt.wantHash)
			}
		})
	}
}

func TestDigest_A2(t *testing.T) {
	tests := []struct {
		name     string
		digest   *auth.Digest
		method   string
		body     []byte
		wantHash bool
	}{
		{
			name: "Basic A2",
			digest: &auth.Digest{
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestMD5,
					URI:       "/dir/index.html",
				},
			},
			method:   http.MethodGet,
			wantHash: true,
		},
		{
			name: "A2 with auth-int",
			digest: &auth.Digest{
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestSHA256,
					URI:       "/dir/index.html",
					QOP:       "auth-int",
				},
			},
			method:   http.MethodPost,
			body:     []byte("test body"),
			wantHash: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.digest.A2(tt.method, tt.body)
			if (got != "") != tt.wantHash {
				t.Errorf("A2() = %v, want hash: %v", got, tt.wantHash)
			}
		})
	}
}

func TestDigest_Response(t *testing.T) {
	tests := []struct {
		name   string
		digest *auth.Digest
		A1     string
		A2     string
		want   string
	}{
		{
			name: "Response without QOP",
			digest: &auth.Digest{
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestMD5,
					Nonce:     "nonce",
				},
			},
			A1:   "a1hash",
			A2:   "a2hash",
			want: "response",
		},
		{
			name: "Response with QOP",
			digest: &auth.Digest{
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestSHA256,
					Nonce:     "nonce",
					QOP:       "auth",
					CNonce:    "cnonce",
					NC:        "00000001",
				},
			},
			A1:   "a1hash",
			A2:   "a2hash",
			want: "response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.digest.Response(tt.A1, tt.A2); got == "" {
				t.Error("Response() returned empty string")
			}
		})
	}
}

//gocyclo:ignore
func TestDigest_Header(t *testing.T) {
	tests := []struct {
		name       string
		digest     *auth.Digest
		method     string
		url        *url.URL
		body       []byte
		wantHeader string
		wantErr    bool
		utf8       bool
	}{
		{
			name: "Basic auth header without qop",
			digest: &auth.Digest{
				Username: "Mufasa",
				Password: "Circle of Life",
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestMD5,
					Realm:     "testrealm@host.com",
					URI:       "/dir/index.html",
					Nonce:     "dcd98b7102dd2f0e8b11d0f600bfb0c093",
					UserHash:  false,
				},
			},
			method: http.MethodGet,
			url:    &url.URL{Path: "/dir/index.html"},
			body:   nil,
		},
		{
			name: "Basic auth header sess without qop",
			digest: &auth.Digest{
				Username: "Mufasa",
				Password: "Circle of Life",
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestSHA256_SESS,
					Realm:     "testrealm@host.com",
					URI:       "/dir/index.html",
					Nonce:     "dcd98b7102dd2f0e8b11d0f600bfb0c093",
					UserHash:  false,
				},
			},
			method: http.MethodGet,
			url:    &url.URL{Path: "/dir/index.html"},
			body:   nil,
		},
		{
			name: "Basic auth header with Opaque",
			digest: &auth.Digest{
				Username: "Mufasa",
				Password: "Circle of Life",
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestSHA256,
					Realm:     "testrealm@host.com",
					URI:       "/dir/index.html",
					Nonce:     "dcd98b7102dd2f0e8b11d0f600bfb0c093",
					UserHash:  false,
					Opaque:    "0a4f113b",
				},
			},
			method: http.MethodGet,
			url:    &url.URL{Path: "/dir/index.html"},
			body:   nil,
		},
		{
			name: "Auth header with qop=auth",
			digest: &auth.Digest{
				Username: "Mufasa",
				Password: "Circle of Life",
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestMD5,
					Realm:     "testrealm@host.com",
					URI:       "/dir/index.html",
					QOP:       "auth",
					Nonce:     "dcd98b7102dd2f0e8b11d0f600bfb0c093",
					CNonce:    "0a4f113b",
					NC:        "00000001",
					UserHash:  false,
				},
			},
			method: http.MethodGet,
			url:    &url.URL{Path: "/dir/index.html"},
			body:   nil,
		},
		{
			name: "Auth header with qop=auth-int",
			digest: &auth.Digest{
				Username: "Mufasa",
				Password: "Circle of Life",
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestSHA256,
					Realm:     "testrealm@host.com",
					URI:       "/dir/index.html",
					QOP:       "auth-int",
					Nonce:     "dcd98b7102dd2f0e8b11d0f600bfb0c093",
					CNonce:    "0a4f113b",
					NC:        "00000001",
					UserHash:  true,
				},
			},
			method: http.MethodPost,
			url:    &url.URL{Path: "/dir/index.html"},
			body:   []byte("test body"),
		},
		{
			name: "Non-ASCII username",
			digest: &auth.Digest{
				Username: "测试用户", // Chinese characters
				Password: "testpass",
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestSHA256,
					Realm:     "testrealm",
					URI:       "/test",
					UserHash:  false,
				},
			},
			method: http.MethodGet,
			url:    &url.URL{Path: "/test"},
			body:   nil,
			utf8:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headerName, headerValue, err := tt.digest.Header(tt.method, tt.url, tt.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("Header() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if headerName != auth.DigestHeaderName {
				t.Errorf("Header() name = %v, want %v", headerName, auth.DigestHeaderName)
			}

			if !strings.HasPrefix(headerValue, auth.DigestValuePrefix) {
				t.Errorf("Header() value prefix = %v, want prefix %v", headerValue, auth.DigestValuePrefix)
			}

			// Check required fields are present
			requiredFields := []string{
				`uri="`,
				`algorithm=`,
				`response="`,
				`userhash=`,
			}

			for _, field := range requiredFields {
				if !strings.Contains(headerValue, field) {
					t.Errorf("Header() value missing required field %q", field)
				}
			}

			if tt.utf8 && !strings.Contains(headerValue, `username*=UTF-8`) {
				t.Errorf("Header() value missing username*=UTF-8 field for UTF-8 username")
			}
			if tt.digest.Parameters.Realm != "" && !strings.Contains(headerValue, `realm="`+tt.digest.Parameters.Realm+`"`) {
				t.Errorf("Header() value missing Realm field")
			}
			if tt.digest.Parameters.Nonce != "" && !strings.Contains(headerValue, `nonce="`+tt.digest.Parameters.Nonce+`"`) {
				t.Errorf("Header() value missing Nonce field")
			}
			if tt.digest.Parameters.QOP != "" && !strings.Contains(headerValue, `qop=`+tt.digest.Parameters.QOP) {
				t.Errorf("Header() value missing QOP field")
			}
			if tt.digest.Parameters.NC != "" && !strings.Contains(headerValue, `nc=`+tt.digest.Parameters.NC) {
				t.Errorf("Header() value missing NC field")
			}
			if tt.digest.Parameters.CNonce != "" && !strings.Contains(headerValue, `cnonce="`+tt.digest.Parameters.CNonce+`"`) {
				t.Errorf("Header() value missing CNonce field")
			}
			if tt.digest.Parameters.Opaque != "" && !strings.Contains(headerValue, `opaque="`+tt.digest.Parameters.Opaque+`"`) {
				t.Errorf("Header() value missing Opaque field")
			}
		})
	}
}
