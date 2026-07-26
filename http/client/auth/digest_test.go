package auth_test

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"gitlab.com/iglou.eu/goulc/hided"
	"gitlab.com/iglou.eu/goulc/http/client/auth"
)

// RFC 7616 section 3.9.1 example values.
// https://www.rfc-editor.org/rfc/rfc7616#section-3.9.1
const (
	rfcUser   = "Mufasa"
	rfcPass   = "Circle of Life"
	rfcRealm  = "http-auth@example.org"
	rfcURI    = "/dir/index.html"
	rfcNonce  = "7ypf/xlj9XXwfDPEoM4URrv/xwf94BcCAzFZH4GiTo0v"
	rfcCNonce = "f2/wE4q74E6zIJEtWaHKaf5wv/H5QzzpXusqGemxURZJ"
	rfcNC     = "00000001"
	rfcOpaque = "FQhe/qaU925kfnzjCev0ciny7QMkPqMAFRtzCUYo5tdS"

	rfcMD5Response    = "8ca523f5e9506fed4657c9700eebdbec"
	rfcSHA256Response = "753927fa0e85d155564e2e272a28d1802ca10daf4496794697cf8db5856cb6c1"

	// Same credentials without qop (RFC 2069 compatibility mode), value
	// computed independently from the implementation.
	rfcNoQOPMD5Response = "7b2cc3b30e75b4777ea31027084363fd"
)

// RFC 7616 section 3.9.2 example values. The response and userhash values
// come from errata 4897: the ones printed in the RFC do not match a
// FIPS 180-4 SHA-512/256 computation.
// https://www.rfc-editor.org/rfc/rfc7616#section-3.9.2
// https://www.rfc-editor.org/errata/eid4897
const (
	rfcUTF8User     = "Jäsøn Doe"
	rfcUTF8Pass     = "Secret, or not?"
	rfcUTF8Realm    = "api@example.org"
	rfcUTF8URI      = "/doe.json"
	rfcUTF8Nonce    = "5TsQWLVdgBdmrQ0XsxbDODV+57QdFR34I9HAbC/RVvkK"
	rfcUTF8CNonce   = "NTg6RKcb9boFIAS3KrFK9BGeh+iDa/sm6jUMp2wds69v"
	rfcUTF8Opaque   = "HRPCssKJSGjCrkzDg8OhwpzCiGPChXYjwrI2QmXDnsOS"
	rfcUTF8Response = "3798d4131c277846293534c3edc11bd8a5e4cdcbff78b05db9d95eeb1cec68a5"
	rfcUTF8UserHash = "793263caabb707a56211940d90411ea4a575adeccb7e360aeb624ed06ece9b0b"
	rfcUTF8UserEnc  = "UTF-8''J%C3%A4s%C3%B8n%20Doe"
)

// rfcDigest returns a Digest loaded with the RFC 7616 section 3.9.1
// example values for the given algorithm.
func rfcDigest(algo auth.DigestAlgo) *auth.Digest {
	return &auth.Digest{
		Username: rfcUser,
		Password: hided.NewString(rfcPass),
		Parameters: auth.DigestParameters{
			Algorithm: algo,
			Realm:     rfcRealm,
			URI:       rfcURI,
			QOP:       auth.DigestQOPAuth,
			Nonce:     rfcNonce,
			CNonce:    rfcCNonce,
			NC:        rfcNC,
			Opaque:    rfcOpaque,
		},
	}
}

func TestNewDigest(t *testing.T) {
	tests := []struct {
		name       string
		username   string
		password   hided.String
		parameters auth.DigestParameters
		wantErr    error
	}{
		{
			name:     "Valid digest",
			username: "Mufasa",
			password: hided.NewString("Circle of Life"),
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
			password: hided.NewString("Circle of Life"),
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
			password: hided.NewString(""),
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
			password: hided.NewString("Circle of Life"),
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
			password: hided.NewString("Circle of Life"),
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
			password: hided.NewString("Circle of Life"),
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
			password: hided.NewString("Circle of Life"),
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
			if got.Password.Reveal() != tt.password.Reveal() {
				t.Errorf("NewDigest().Password = %v, want %v", got.Password, tt.password)
			}
			if got.Parameters != tt.parameters {
				t.Errorf("NewDigest().Parameters = %+v, want %+v", got.Parameters, tt.parameters)
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
func TestDigest_Update(_ *testing.T) {
	d := &auth.Digest{}
	_ = d.Update()
}

func TestDigest_Clone(t *testing.T) {
	original := &auth.Digest{
		Username: "testuser",
		Password: hided.NewString("testpass"),
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
	d, ok := cloned.(*auth.Digest)
	if !ok {
		t.Fatalf("Clone() returned %T, want *auth.Digest", cloned)
	}
	if original == d {
		t.Error("Clone() returned same pointer instead of new instance")
	}

	// Check if all values are the same
	if d.Username != original.Username {
		t.Errorf("Clone().Username = %v, want %v", d.Username, original.Username)
	}
	if d.Password.Reveal() != original.Password.Reveal() {
		t.Errorf("Clone().Password = %v, want %v", d.Password, original.Password)
	}
	if d.Parameters != original.Parameters {
		t.Errorf("Clone().Parameters = %+v, want %+v", d.Parameters, original.Parameters)
	}
}

func TestDigestParameters_Hash(t *testing.T) {
	testData := []byte("test data")
	tests := []struct {
		name      string
		algorithm auth.DigestAlgo
		want      string
		wantErr   error
	}{
		{
			name:      "MD5",
			algorithm: auth.DigestMD5,
			want:      "eb733a00c0c9d336e65691a37ab54293",
		},
		{
			name:      "MD5-SESS",
			algorithm: auth.DigestMD5SESS,
			want:      "eb733a00c0c9d336e65691a37ab54293",
		},
		{
			name:      "SHA-256",
			algorithm: auth.DigestSHA256,
			want:      "916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9",
		},
		{
			name:      "SHA-256-SESS",
			algorithm: auth.DigestSHA256SESS,
			want:      "916f0027a575074ce72a331777c3478d6513f786a591bd892da1a577bf2335f9",
		},
		{
			name:      "SHA-512",
			algorithm: auth.DigestSHA512,
			want:      "0e1e21ecf105ec853d24d728867ad70613c21663a4693074b2a3619c1bd39d66b588c33723bb466c72424e80e3ca63c249078ab347bab9428500e7ee43059d0d",
		},
		{
			name:      "SHA-512-SESS",
			algorithm: auth.DigestSHA512SESS,
			want:      "0e1e21ecf105ec853d24d728867ad70613c21663a4693074b2a3619c1bd39d66b588c33723bb466c72424e80e3ca63c249078ab347bab9428500e7ee43059d0d",
		},
		{
			name:      "SHA-512-256",
			algorithm: auth.DigestSHA512256,
			want:      "9fe875600168548c1954aed4f03974ce06b3e17f03a70980190da2d7ef937a43",
		},
		{
			name:      "SHA-512-256-SESS",
			algorithm: auth.DigestSHA512256SESS,
			want:      "9fe875600168548c1954aed4f03974ce06b3e17f03a70980190da2d7ef937a43",
		},
		{
			name:      "Unknown algorithm",
			algorithm: "invalid",
			want:      "",
			wantErr:   auth.ErrUnknownAlgorithm,
		},
		{
			name:      "Empty algorithm",
			algorithm: "",
			want:      "",
			wantErr:   auth.ErrUnknownAlgorithm,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &auth.DigestParameters{Algorithm: tt.algorithm}
			got, err := d.Hash(testData)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Hash() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Hash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDigest_A1(t *testing.T) {
	tests := []struct {
		name    string
		digest  *auth.Digest
		want    string
		wantErr error
	}{
		{
			name:   "Basic A1",
			digest: rfcDigest(auth.DigestMD5),
			want:   rfcUser + ":" + rfcRealm + ":" + rfcPass,
		},
		{
			name: "Session A1",
			digest: &auth.Digest{
				Username: "Mufasa",
				Password: hided.NewString("Circle of Life"),
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestMD5SESS,
					Realm:     "testrealm@host.com",
					Nonce:     "nonce",
					CNonce:    "cnonce",
				},
			},
			// md5("Mufasa:testrealm@host.com:Circle of Life") joined with
			// the nonce and cnonce values.
			want: "7650d211d93fae2c3f56cdb1f1af23b2:nonce:cnonce",
		},
		{
			name: "Session A1 with unknown algorithm",
			digest: &auth.Digest{
				Username: "Mufasa",
				Password: hided.NewString("Circle of Life"),
				Parameters: auth.DigestParameters{
					Algorithm: "invalid-sess",
					Realm:     "testrealm@host.com",
					Nonce:     "nonce",
					CNonce:    "cnonce",
				},
			},
			want:    "",
			wantErr: auth.ErrUnknownAlgorithm,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.digest.A1()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("A1() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("A1() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDigest_A2(t *testing.T) {
	tests := []struct {
		name    string
		digest  *auth.Digest
		method  string
		body    []byte
		want    string
		wantErr error
	}{
		{
			name: "Basic A2",
			digest: &auth.Digest{
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestMD5,
					URI:       "/dir/index.html",
				},
			},
			method: http.MethodGet,
			want:   "GET:/dir/index.html",
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
			method: http.MethodPost,
			body:   []byte("test body"),
			// sha256("test body") appended to the method and URI.
			want: "POST:/dir/index.html:" +
				"63efb315ed71cc7e5a1fc202434bb3aec2091e7838707e148a017faebb7464fe",
		},
		{
			name: "A2 with auth-int and unknown algorithm",
			digest: &auth.Digest{
				Parameters: auth.DigestParameters{
					Algorithm: "invalid",
					URI:       "/dir/index.html",
					QOP:       "auth-int",
				},
			},
			method:  http.MethodPost,
			body:    []byte("test body"),
			want:    "",
			wantErr: auth.ErrUnknownAlgorithm,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.digest.A2(tt.method, tt.body)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("A2() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("A2() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDigest_Response(t *testing.T) {
	noQOP := rfcDigest(auth.DigestMD5)
	noQOP.Parameters.QOP = ""
	noQOP.Parameters.CNonce = ""
	noQOP.Parameters.NC = ""

	tests := []struct {
		name    string
		digest  *auth.Digest
		a1      string
		a2      string
		want    string
		wantErr error
	}{
		{
			name:   "RFC 7616 vector with qop and MD5",
			digest: rfcDigest(auth.DigestMD5),
			a1:     rfcUser + ":" + rfcRealm + ":" + rfcPass,
			a2:     "GET:" + rfcURI,
			want:   rfcMD5Response,
		},
		{
			name:   "RFC 7616 vector with qop and SHA-256",
			digest: rfcDigest(auth.DigestSHA256),
			a1:     rfcUser + ":" + rfcRealm + ":" + rfcPass,
			a2:     "GET:" + rfcURI,
			want:   rfcSHA256Response,
		},
		{
			name:   "Without qop, RFC 2069 compatibility",
			digest: noQOP,
			a1:     rfcUser + ":" + rfcRealm + ":" + rfcPass,
			a2:     "GET:" + rfcURI,
			want:   rfcNoQOPMD5Response,
		},
		{
			name: "Unknown algorithm",
			digest: &auth.Digest{
				Parameters: auth.DigestParameters{
					Algorithm: "invalid",
					Nonce:     "nonce",
				},
			},
			a1:      "a1",
			a2:      "a2",
			want:    "",
			wantErr: auth.ErrUnknownAlgorithm,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.digest.Response(tt.a1, tt.a2)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Response() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Response() = %v, want %v", got, tt.want)
			}
		})
	}
}

//gocyclo:ignore
func TestDigest_Header(t *testing.T) {
	noQOP := rfcDigest(auth.DigestMD5)
	noQOP.Parameters.QOP = ""
	noQOP.Parameters.CNonce = ""
	noQOP.Parameters.NC = ""

	utf8Digest := func(userHash bool) *auth.Digest {
		return &auth.Digest{
			Username: rfcUTF8User,
			Password: hided.NewString(rfcUTF8Pass),
			Parameters: auth.DigestParameters{
				Algorithm: auth.DigestSHA512256,
				Realm:     rfcUTF8Realm,
				URI:       rfcUTF8URI,
				QOP:       auth.DigestQOPAuth,
				Nonce:     rfcUTF8Nonce,
				CNonce:    rfcUTF8CNonce,
				NC:        rfcNC,
				Opaque:    rfcUTF8Opaque,
				UserHash:  userHash,
			},
		}
	}

	tests := []struct {
		name         string
		digest       *auth.Digest
		method       string
		body         []byte
		wantResponse string
		wantUsername string
	}{
		{
			name:         "RFC 7616 3.9.1 with MD5",
			digest:       rfcDigest(auth.DigestMD5),
			method:       http.MethodGet,
			wantResponse: `response="` + rfcMD5Response + `"`,
			wantUsername: `username="` + rfcUser + `"`,
		},
		{
			name:         "RFC 7616 3.9.1 with SHA-256",
			digest:       rfcDigest(auth.DigestSHA256),
			method:       http.MethodGet,
			wantResponse: `response="` + rfcSHA256Response + `"`,
			wantUsername: `username="` + rfcUser + `"`,
		},
		{
			name:         "RFC 7616 3.9.2 with SHA-512-256 and userhash",
			digest:       utf8Digest(true),
			method:       http.MethodGet,
			wantResponse: `response="` + rfcUTF8Response + `"`,
			wantUsername: `username="` + rfcUTF8UserHash + `"`,
		},
		{
			name:         "RFC 7616 3.9.2 with SHA-512-256 and username*",
			digest:       utf8Digest(false),
			method:       http.MethodGet,
			wantResponse: `response="` + rfcUTF8Response + `"`,
			wantUsername: `username*=` + rfcUTF8UserEnc,
		},
		{
			name:         "Without qop, RFC 2069 compatibility",
			digest:       noQOP,
			method:       http.MethodGet,
			wantResponse: `response="` + rfcNoQOPMD5Response + `"`,
			wantUsername: `username="` + rfcUser + `"`,
		},
		{
			name: "Username with DEL goes through extended encoding",
			digest: &auth.Digest{
				// DEL (0x7f) is a control character, not printable ASCII,
				// so the username must not be sent in the plain form.
				Username: "del\x7fuser",
				Password: hided.NewString("testpass"),
				Parameters: auth.DigestParameters{
					Algorithm: auth.DigestSHA256,
					Realm:     "testrealm",
					URI:       "/test",
					Nonce:     "testnonce",
				},
			},
			method:       http.MethodGet,
			wantUsername: `username*=UTF-8''del%7Fuser`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headerName, headerValue, err := tt.digest.Header(
				tt.method, &url.URL{Path: tt.digest.Parameters.URI}, tt.body)
			if err != nil {
				t.Fatalf("Header() error = %v", err)
			}

			if headerName != auth.DigestHeaderName {
				t.Errorf("Header() name = %v, want %v", headerName, auth.DigestHeaderName)
			}
			if !strings.HasPrefix(headerValue, auth.DigestValuePrefix) {
				t.Errorf("Header() value = %v, want prefix %v", headerValue, auth.DigestValuePrefix)
			}

			if tt.wantResponse != "" && !strings.Contains(headerValue, tt.wantResponse) {
				t.Errorf("Header() value = %v, want it to contain %v", headerValue, tt.wantResponse)
			}
			if !strings.Contains(headerValue, tt.wantUsername) {
				t.Errorf("Header() value = %v, want it to contain %v", headerValue, tt.wantUsername)
			}

			p := tt.digest.Parameters
			if !strings.Contains(headerValue, `uri="`+p.URI+`"`) {
				t.Errorf("Header() value missing URI field")
			}
			if !strings.Contains(headerValue, `realm="`+p.Realm+`"`) {
				t.Errorf("Header() value missing Realm field")
			}
			if !strings.Contains(headerValue, `nonce="`+p.Nonce+`"`) {
				t.Errorf("Header() value missing Nonce field")
			}
			if p.QOP != "" && !strings.Contains(headerValue, `qop=`+p.QOP) {
				t.Errorf("Header() value missing QOP field")
			}
			if p.QOP == "" && strings.Contains(headerValue, `qop=`) {
				t.Errorf("Header() value must not carry a qop field without qop")
			}
			if p.NC != "" && !strings.Contains(headerValue, `nc=`+p.NC) {
				t.Errorf("Header() value missing NC field")
			}
			if p.CNonce != "" && !strings.Contains(headerValue, `cnonce="`+p.CNonce+`"`) {
				t.Errorf("Header() value missing CNonce field")
			}
			if p.CNonce == "" && strings.Contains(headerValue, `cnonce=`) {
				t.Errorf("Header() value must not carry a cnonce field without qop")
			}
			if p.Opaque != "" && !strings.Contains(headerValue, `opaque="`+p.Opaque+`"`) {
				t.Errorf("Header() value missing Opaque field")
			}
		})
	}
}

func TestDigest_Header_Errors(t *testing.T) {
	noCNonce := rfcDigest(auth.DigestMD5)
	noCNonce.Parameters.CNonce = ""

	noNC := rfcDigest(auth.DigestMD5)
	noNC.Parameters.NC = ""

	unknownQOP := rfcDigest(auth.DigestMD5)
	unknownQOP.Parameters.QOP = "invalid"

	badUserHash := rfcDigest("invalid")
	badUserHash.Parameters.QOP = ""
	badUserHash.Parameters.UserHash = true

	badSess := rfcDigest("invalid-sess")
	badSess.Parameters.QOP = ""

	badAuthInt := rfcDigest("invalid")
	badAuthInt.Parameters.QOP = auth.DigestQOPAuthInt

	badAuth := rfcDigest("invalid")

	tests := []struct {
		name    string
		digest  *auth.Digest
		wantErr error
	}{
		{
			name:    "qop without cnonce",
			digest:  noCNonce,
			wantErr: auth.ErrNoCNonce,
		},
		{
			name:    "qop without nc",
			digest:  noNC,
			wantErr: auth.ErrNoNC,
		},
		{
			name:    "unsupported qop value",
			digest:  unknownQOP,
			wantErr: auth.ErrUnknownQOP,
		},
		{
			name:    "unknown algorithm with userhash",
			digest:  badUserHash,
			wantErr: auth.ErrUnknownAlgorithm,
		},
		{
			name:    "unknown session algorithm",
			digest:  badSess,
			wantErr: auth.ErrUnknownAlgorithm,
		},
		{
			name:    "unknown algorithm with auth-int",
			digest:  badAuthInt,
			wantErr: auth.ErrUnknownAlgorithm,
		},
		{
			name:    "unknown algorithm with auth",
			digest:  badAuth,
			wantErr: auth.ErrUnknownAlgorithm,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headerName, headerValue, err := tt.digest.Header(
				http.MethodGet, &url.URL{Path: tt.digest.Parameters.URI}, nil)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Header() error = %v, wantErr %v", err, tt.wantErr)
			}
			if headerName != "" || headerValue != "" {
				t.Errorf("Header() = (%q, %q), want empty values on error",
					headerName, headerValue)
			}
		})
	}
}
