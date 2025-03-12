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

package auth

import (
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"net/url"
	"strconv"
	"strings"

	"gitlab.com/iglou.eu/goulc/ascii"
)

// DigestAlgo represents the supported hash algorithms for HTTP
// Digest Authentication. It includes standard algorithms from RFC7616
// and adds SHA512 for future compatibility. See RFC7616 Section 6.1
// at https://datatracker.ietf.org/doc/html/rfc7616#section-6.1
type DigestAlgo string

const (
	// DigestName is the identifier for this authentication method
	DigestName = "auth.Digest"
	// DigestSeparator is used to separate components in digest calculation
	DigestSeparator = ":"
	// DigestHeaderName is the HTTP header name for authentication
	DigestHeaderName = "Authorization"
	// DigestValuePrefix is the prefix for the digest authentication value
	DigestValuePrefix = "Digest "

	// Quality of Protection (QOP) options

	// DigestQOPAuth Authentication only
	DigestQOPAuth = "auth"
	// DigestQOPAuthInt Authentication with integrity protection
	DigestQOPAuthInt = "auth-int"

	// Standard hash algorithms

	DigestMD5       DigestAlgo = "md5"
	DigestSHA256    DigestAlgo = "sha-256"
	DigestSHA512    DigestAlgo = "sha-512" // Future-proof addition
	DigestSHA512256 DigestAlgo = "sha-512-256"

	// Session variants of hash algorithms

	DigestMD5SESS       DigestAlgo = "md5-sess"
	DigestSHA256SESS    DigestAlgo = "sha-256-sess"
	DigestSHA512SESS    DigestAlgo = "sha-512-sess"
	DigestSHA512256SESS DigestAlgo = "sha-512-256-sess"
)

// Verify Digest implements Authenticator interface
var _ Authenticator = &Digest{}

// Digest implements the HTTP Digest Authentication scheme as defined in
// RFC7616. It provides both standard authentication and session-based variants.
type Digest struct {
	// Username for authentication
	Username string
	// Password for authentication
	Password string

	// Parameters contains all the digest authentication parameters
	Parameters DigestParameters
}

// DigestParameters contains all the fields required for digest
// authentication as defined in RFC7616 Section 3.4
// at https://datatracker.ietf.org/doc/html/rfc7616#section-3.4
type DigestParameters struct {
	// Algorithm specifies the hash algorithm to use
	Algorithm DigestAlgo

	// Realm indicates the protection space
	Realm string
	// URI is the request URI
	URI string
	// QOP (Quality of Protection) can be "auth" or "auth-int"
	QOP string
	// Nonce is the server-specified data string
	Nonce string
	// CNonce is the client-specified data string
	CNonce string
	// NC (Nonce Count) is the hexadecimal count of requests
	NC string
	// UserHash indicates if the username should be hashed
	UserHash bool
	// Opaque is the server-specified data string
	Opaque string
}

// digestValues represents a collection of digest authentication header values.
type digestValues []digestValue

// digestValue represents a single key-value pair in the digest
// authentication header. It includes metadata about how
// the value should be formatted (quoted and/or with asterisk).
type digestValue struct {
	key      string // The key name in the header
	value    string // The value associated with the key
	quoted   bool   // If the value should be quoted in the header
	asterisk bool   // If the key should have a * suffix (UTF-8 encoding)
}

// marshal converts the digest values into a properly formatted string
// for use in an HTTP header.
func (d *digestValues) marshal() string {
	// Pre-allocate the slice to avoid reallocations
	entries := make([]string, 0, len(*d))

	// Process each digest value
	for _, v := range *d {
		// Skip empty values as per RFC
		if v.value == "" {
			continue
		}

		// Pre-calculate the entry size
		var entry strings.Builder
		entry.Grow(len(v.key) + len(v.value) + 4) // +3 for '=', '*' and quotes

		// Build the entry string
		entry.WriteString(v.key)
		if v.asterisk {
			entry.WriteRune('*')
		}
		entry.WriteRune('=')
		if v.quoted {
			entry.WriteRune('"')
		}
		entry.WriteString(v.value)
		if v.quoted {
			entry.WriteRune('"')
		}

		entries = append(entries, entry.String())
	}

	// Join all entries with comma and space
	return strings.Join(entries, ", ")
}

// NewDigest creates a new Digest authentication instance with
// the provided credentials and parameters.
//
// It performs a minimal validation of required fields:
// - Algorithm must be supported
// - Username must not be empty
// - Password must not be empty
// - Realm must not be empty
// - Nonce must not be empty
// - URI must not be empty
//
// Returns an error if any of the required fields are invalid or missing.
func NewDigest(
	username, password string, parameters DigestParameters,
) (Digest, error) {
	if username == "" {
		return Digest{}, ErrNoUserID
	}

	if password == "" {
		return Digest{}, ErrNoPassword
	}

	if parameters.Hash([]byte("")) == ErrUnknownAlgorithm.Error() {
		return Digest{}, ErrUnknownAlgorithm
	}

	if parameters.Realm == "" {
		return Digest{}, ErrNoRealm
	}

	if parameters.Nonce == "" {
		return Digest{}, ErrNoNonce
	}

	if parameters.URI == "" {
		return Digest{}, ErrNoURI
	}

	return Digest{
		Username: username,
		Password: password,
		Parameters: DigestParameters{
			Algorithm: parameters.Algorithm,
			Realm:     parameters.Realm,
			URI:       parameters.URI,
			QOP:       parameters.QOP,
			Nonce:     parameters.Nonce,
			CNonce:    parameters.CNonce,
			NC:        parameters.NC,
			UserHash:  parameters.UserHash,
			Opaque:    parameters.Opaque,
		},
	}, nil
}

// Name returns the identifier for this authentication method.
func (_ *Digest) Name() string {
	return DigestName
}

// Update implements the Authenticator interface.
// This method is a no-op as there is no state to update.
func (_ *Digest) Update() error {
	return nil
}

// Header generates the HTTP Authorization header for Digest authentication.
// It follows RFC7616 specifications while maintaining backwards compatibility
// with RFC2069.
//
// The method constructs the header by:
// 1. Computing the response using A1 and A2 values
// 2. Building the header with all required fields
// 3. Handling username encoding
func (d *Digest) Header(method string, _ *url.URL, body []byte,
) (headerKey, headerValue string, err error) {
	a1 := d.A1()
	a2 := d.A2(method, body)
	response := d.Parameters.Hash([]byte(d.Response(a1, a2)))

	// Formating the Authorization Header Field Defined under RFC7616-3.4
	// at https://datatracker.ietf.org/doc/html/rfc7616#section-3.4
	digestValues := digestValues{
		{
			key:    "uri",
			value:  d.Parameters.URI,
			quoted: true,
		},
		{
			key:   "algorithm",
			value: string(d.Parameters.Algorithm),
		},
		{
			key:    "response",
			value:  response,
			quoted: true,
		},
		{
			key:    "realm",
			value:  d.Parameters.Realm,
			quoted: true,
		},
		{
			key:    "nonce",
			value:  d.Parameters.Nonce,
			quoted: true,
		},
		{
			key:   "nc",
			value: d.Parameters.NC,
		},
		{
			key:    "cnonce",
			value:  d.Parameters.CNonce,
			quoted: true,
		},
		{
			key:   "qop",
			value: d.Parameters.QOP,
		},
		{
			key:    "opaque",
			value:  d.Parameters.Opaque,
			quoted: true,
		},
		{
			key:   "userhash",
			value: strconv.FormatBool(d.Parameters.UserHash),
		},
	}

	// User hash or UTF-8 username declaration header
	// Defined under RFC7616-3.4.4
	// at https://datatracker.ietf.org/doc/html/rfc7616#section-3.4.4
	// And under RFC7616-4
	// at https://datatracker.ietf.org/doc/html/rfc7616#section-4
	switch {
	case d.Parameters.UserHash:
		digestValues = append(digestValues, digestValue{
			key: "username",
			value: d.Parameters.Hash(
				[]byte(d.Username + `:` + d.Parameters.Realm)),
			quoted: true,
		})
	case ascii.IsPrintable(d.Username):
		digestValues = append(digestValues, digestValue{
			key:    "username",
			value:  d.Username,
			quoted: true,
		})
	default:
		digestValues = append(digestValues, digestValue{
			key:      "username",
			value:    "UTF-8''" + url.QueryEscape(d.Username),
			asterisk: true,
		})
	}

	return DigestHeaderName, DigestValuePrefix + digestValues.marshal(), nil
}

// Clone creates a deep copy of the instance.
// This ensures that modifications don't affect the original instance.
func (d *Digest) Clone() Authenticator {
	return &Digest{
		Username:   d.Username,
		Password:   d.Password,
		Parameters: d.Parameters,
	}
}

// A1 computes the A1 value as specified in RFC7616 section 3.4.2.
// at https://datatracker.ietf.org/doc/html/rfc7616#section-3.4.2
// For session-based algorithms (-sess suffix), it includes the nonce
// and cnonce values. Returns the computed A1 value used as secret Keyed.
func (d *Digest) A1() string {
	a1 := strings.Join([]string{
		d.Username,
		d.Parameters.Realm,
		d.Password,
	}, DigestSeparator)

	if strings.HasSuffix(string(d.Parameters.Algorithm), "-sess") {
		return strings.Join([]string{
			d.Parameters.Hash([]byte(a1)),
			d.Parameters.Nonce,
			d.Parameters.CNonce,
		}, DigestSeparator)
	}

	return a1
}

// A2 computes the A2 value as specified in RFC7616 section 3.4.3.
// at https://datatracker.ietf.org/doc/html/rfc7616#section-3.4.3
// When using auth-int quality of protection, it includes a hash of
// the request body. Returns the computed A2 value used in response generation.
func (d *Digest) A2(method string, body []byte) string {
	if d.Parameters.QOP == DigestQOPAuthInt {
		return strings.Join([]string{
			method,
			d.Parameters.URI,
			d.Parameters.Hash(body),
		}, DigestSeparator)
	}

	return strings.Join([]string{
		method,
		d.Parameters.URI,
	}, DigestSeparator)
}

// Response generates the digest response according to RFC7616 section 3.4.1
// at https://datatracker.ietf.org/doc/html/rfc7616#section-3.4.1
// It supports both standard authentication and quality of protection modes.
// Note: Support the deprecated RFC2069 section 2.1.2 backwards compatibility
// at https://datatracker.ietf.org/doc/html/rfc2069#section-2.1.2
// Returns the response string used in the Authorization header.
func (d *Digest) Response(a1, a2 string) string {
	if d.Parameters.QOP == DigestQOPAuth ||
		d.Parameters.QOP == DigestQOPAuthInt {
		return strings.Join([]string{
			d.Parameters.Hash([]byte(a1)), // The secret Keyed Digest
			d.Parameters.Nonce,
			d.Parameters.NC,
			d.Parameters.CNonce,
			d.Parameters.QOP,
			d.Parameters.Hash([]byte(a2)),
		}, DigestSeparator)
	}

	return strings.Join([]string{
		d.Parameters.Hash([]byte(a1)), // The secret Keyed Digest
		d.Parameters.Nonce,
		d.Parameters.Hash([]byte(a2)),
	}, DigestSeparator)
}

// Hash computes the digest hash value using the specified algorithm.
// It supports multiple hash algorithms as defined in RFC7616 and RFC2617:
// - MD5 (RFC2069 and RFC2617)
// - SHA-256 (RFC7616)
// - SHA-512-256 (RFC7616)
// - SHA-512 (future-proof extension)
//
// Returns the hexadecimal string representation of the hash.
// If the algorithm is not supported, it returns
// the ErrUnknownAlgorithm error message.
func (d *DigestParameters) Hash(s []byte) string {
	switch d.Algorithm {
	case DigestSHA256, DigestSHA256SESS:
		sum := sha256.Sum256(s)
		return hex.EncodeToString(sum[:])
	case DigestSHA512256, DigestSHA512256SESS:
		sum := sha512.Sum512_256(s)
		return hex.EncodeToString(sum[:])
	case DigestSHA512, DigestSHA512SESS:
		sum := sha512.Sum512(s)
		return hex.EncodeToString(sum[:])
	case DigestMD5, DigestMD5SESS:
		sum := md5.Sum(s)
		return hex.EncodeToString(sum[:])
	}

	return ErrUnknownAlgorithm.Error()
}
