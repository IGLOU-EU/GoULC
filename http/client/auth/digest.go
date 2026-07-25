/*
 * Copyright 2025 Adrien Kara
 *
 * This file is part of GoULC.
 *
 * This is free software: you can redistribute it and/or modify
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
 * SPDX-License-Identifier: LGPL-3.0-or-later
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
	"gitlab.com/iglou.eu/goulc/hided"
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
var _ Authenticator = (*Digest)(nil)

// Digest implements the HTTP Digest Authentication scheme as defined in
// RFC7616. It provides both standard authentication and session-based
// variants. The password is held as a hided.String so fmt verbs and
// marshaling cannot leak it.
type Digest struct {
	// Username for authentication
	Username string
	// Password for authentication, in hidden mode
	Password hided.String

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

// marshal renders the values as a complete Digest Authorization header
// value. A single pre-sized buffer keeps it to one allocation per header
// instead of one per entry.
func (d digestValues) marshal() string {
	// Worst case per entry: '*', '=', two quotes and the ", " separator
	const entryOverhead = 6

	var sb strings.Builder

	size := len(DigestValuePrefix)
	for _, v := range d {
		size += len(v.key) + len(v.value) + entryOverhead
	}
	sb.Grow(size)

	sb.WriteString(DigestValuePrefix)
	first := true
	for _, v := range d {
		// Skip empty values as per RFC
		if v.value == "" {
			continue
		}

		if !first {
			sb.WriteString(", ")
		}
		first = false

		sb.WriteString(v.key)
		if v.asterisk {
			sb.WriteByte('*')
		}
		sb.WriteByte('=')
		if v.quoted {
			sb.WriteByte('"')
		}
		sb.WriteString(v.value)
		if v.quoted {
			sb.WriteByte('"')
		}
	}

	return sb.String()
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
// The per-request parameters tied to QOP (CNonce, NC) are validated later,
// by Header, so they can be set or refreshed after construction.
//
// Returns an error if any of the required fields are invalid or missing.
func NewDigest(
	username string, password hided.String, parameters DigestParameters,
) (Digest, error) {
	if username == "" {
		return Digest{}, ErrNoUserID
	}

	if password.IsEmpty() {
		return Digest{}, ErrNoPassword
	}

	// Probe the algorithm once so an unsupported value fails at
	// construction instead of surfacing on the first request
	if _, err := parameters.Hash(nil); err != nil {
		return Digest{}, err
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
		Username:   username,
		Password:   password,
		Parameters: parameters,
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
// 1. Validating the QOP related parameters
// 2. Computing the username entry and the response value
// 3. Building the header with all required fields
//
// It returns an error if the algorithm is not supported, if the QOP value
// is not supported, or if QOP is set without CNonce or NC, which RFC7616
// section 3.4 makes mandatory alongside qop.
func (d *Digest) Header(method string, _ *url.URL, body []byte,
) (headerKey, headerValue string, err error) {
	if err = d.Parameters.validate(); err != nil {
		return "", "", err
	}

	username, err := d.usernameValue()
	if err != nil {
		return "", "", err
	}

	a1, err := d.A1()
	if err != nil {
		return "", "", err
	}

	a2, err := d.A2(method, body)
	if err != nil {
		return "", "", err
	}

	response, err := d.Response(a1, a2)
	if err != nil {
		return "", "", err
	}

	// Formating the Authorization Header Field Defined under RFC7616-3.4
	// at https://datatracker.ietf.org/doc/html/rfc7616#section-3.4
	values := digestValues{
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
		username,
	}

	return DigestHeaderName, values.marshal(), nil
}

// usernameValue renders the username entry of the Authorization header,
// hashed, plain, or extended notation, as defined under RFC7616-3.4.4
// at https://datatracker.ietf.org/doc/html/rfc7616#section-3.4.4
// and under RFC7616-4
// at https://datatracker.ietf.org/doc/html/rfc7616#section-4
func (d *Digest) usernameValue() (digestValue, error) {
	switch {
	case d.Parameters.UserHash:
		hash, err := d.Parameters.Hash(
			[]byte(d.Username + DigestSeparator + d.Parameters.Realm))
		if err != nil {
			return digestValue{}, err
		}

		return digestValue{key: "username", value: hash, quoted: true}, nil
	case ascii.IsPrintable(d.Username):
		return digestValue{
			key: "username", value: d.Username, quoted: true,
		}, nil
	default:
		// RFC5987 percent-encodes a space as "%20" while QueryEscape
		// emits "+", swap it so the value decodes to the original name
		value := "UTF-8''" + strings.ReplaceAll(
			url.QueryEscape(d.Username), "+", "%20")

		return digestValue{
			key: "username", value: value, asterisk: true,
		}, nil
	}
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
// and cnonce values. Returns the computed A1 value used as secret Keyed,
// or an error if the algorithm is not supported.
func (d *Digest) A1() (string, error) {
	a1 := strings.Join([]string{
		d.Username,
		d.Parameters.Realm,
		d.Password.Reveal(),
	}, DigestSeparator)

	if !strings.HasSuffix(string(d.Parameters.Algorithm), "-sess") {
		return a1, nil
	}

	hash, err := d.Parameters.Hash([]byte(a1))
	if err != nil {
		return "", err
	}

	return strings.Join([]string{
		hash,
		d.Parameters.Nonce,
		d.Parameters.CNonce,
	}, DigestSeparator), nil
}

// A2 computes the A2 value as specified in RFC7616 section 3.4.3.
// at https://datatracker.ietf.org/doc/html/rfc7616#section-3.4.3
// When using auth-int quality of protection, it includes a hash of
// the request body. Returns the computed A2 value used in response
// generation, or an error if the algorithm is not supported.
func (d *Digest) A2(method string, body []byte) (string, error) {
	if d.Parameters.QOP != DigestQOPAuthInt {
		return method + DigestSeparator + d.Parameters.URI, nil
	}

	hash, err := d.Parameters.Hash(body)
	if err != nil {
		return "", err
	}

	return strings.Join([]string{
		method,
		d.Parameters.URI,
		hash,
	}, DigestSeparator), nil
}

// Response generates the digest response according to RFC7616 section 3.4.1
// at https://datatracker.ietf.org/doc/html/rfc7616#section-3.4.1
// It supports both standard authentication and quality of protection modes.
// Note: Support the deprecated RFC2069 section 2.1.2 backwards compatibility
// at https://datatracker.ietf.org/doc/html/rfc2069#section-2.1.2
// Returns the hexadecimal response value used in the Authorization header,
// or an error if the algorithm is not supported.
func (d *Digest) Response(a1, a2 string) (string, error) {
	ha1, err := d.Parameters.Hash([]byte(a1))
	if err != nil {
		return "", err
	}

	// The ha1 hash just proved the algorithm is supported, so hashing
	// with the same algorithm cannot fail anymore
	ha2, _ := d.Parameters.Hash([]byte(a2))

	if d.Parameters.QOP == DigestQOPAuth ||
		d.Parameters.QOP == DigestQOPAuthInt {
		return d.Parameters.Hash([]byte(strings.Join([]string{
			ha1, // The secret Keyed Digest
			d.Parameters.Nonce,
			d.Parameters.NC,
			d.Parameters.CNonce,
			d.Parameters.QOP,
			ha2,
		}, DigestSeparator)))
	}

	return d.Parameters.Hash([]byte(strings.Join([]string{
		ha1, // The secret Keyed Digest
		d.Parameters.Nonce,
		ha2,
	}, DigestSeparator)))
}

// validate enforces the RFC7616 section 3.4 requirement that a client
// nonce and a nonce count accompany any request protected by QOP, and
// that the QOP value is one the response computation supports.
func (d *DigestParameters) validate() error {
	switch d.QOP {
	case "":
		// No QOP means the deprecated RFC2069 compatibility mode,
		// which uses neither CNonce nor NC
		return nil
	case DigestQOPAuth, DigestQOPAuthInt:
		if d.CNonce == "" {
			return ErrNoCNonce
		}

		if d.NC == "" {
			return ErrNoNC
		}

		return nil
	default:
		return ErrUnknownQOP
	}
}

// Hash computes the digest hash value using the specified algorithm.
// It supports multiple hash algorithms as defined in RFC7616 and RFC2617:
// - MD5 (RFC2069 and RFC2617)
// - SHA-256 (RFC7616)
// - SHA-512-256 (RFC7616)
// - SHA-512 (future-proof extension)
//
// Returns the hexadecimal string representation of the hash, or
// ErrUnknownAlgorithm if the algorithm is not supported.
func (d *DigestParameters) Hash(s []byte) (string, error) {
	switch d.Algorithm {
	case DigestSHA256, DigestSHA256SESS:
		sum := sha256.Sum256(s)
		return hex.EncodeToString(sum[:]), nil
	case DigestSHA512256, DigestSHA512256SESS:
		sum := sha512.Sum512_256(s)
		return hex.EncodeToString(sum[:]), nil
	case DigestSHA512, DigestSHA512SESS:
		sum := sha512.Sum512(s)
		return hex.EncodeToString(sum[:]), nil
	case DigestMD5, DigestMD5SESS:
		sum := md5.Sum(s)
		return hex.EncodeToString(sum[:]), nil
	default:
		return "", ErrUnknownAlgorithm
	}
}
