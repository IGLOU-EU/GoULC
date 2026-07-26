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

package oauth2

import (
	"encoding/json"
	"net/http"
	"time"

	"gitlab.com/iglou.eu/goulc/contract"
	"gitlab.com/iglou.eu/goulc/hided"
	"gitlab.com/iglou.eu/goulc/http/client"
)

const (
	// ResponseName is the identifier for Response Unmarshaler
	ResponseName = "oauth2.Response"
)

// Verify interface conformance at compile time
var (
	_ client.Unmarshaler = (*Response)(nil)
	_ contract.Emptier   = ErrorResponse{}
)

// TokenResponse represents successful access token response
// RFC 6749 §5.1: https://www.rfc-editor.org/rfc/rfc6749#section-5.1
type TokenResponse struct {
	Token     hided.String `json:"access_token"`
	TokenType string       `json:"token_type"`

	// ExpiresIn is the token lifetime in seconds, as defined by
	// RFC 6749 §5.1. When the server omits it, the token is treated
	// as already expired and refreshed on every request.
	ExpiresIn int64 `json:"expires_in"`

	RefreshToken hided.String `json:"refresh_token"`
	Scope        string       `json:"scope"`

	// ExpireAt is the absolute expiry instant, computed locally from
	// ExpiresIn when the token is issued. It is not part of the wire
	// response.
	ExpireAt time.Time `json:"-"`
}

// ErrorResponse represents error response
// RFC 6749 §5.2: https://www.rfc-editor.org/rfc/rfc6749#section-5.2
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorURI         string `json:"error_uri"`
}

// IsEmpty reports whether the response carries no error. The error code
// is required on OAuth2 error responses (RFC 6749 §5.2), so an empty
// code means the server did not return an error.
func (e ErrorResponse) IsEmpty() bool {
	return e.Error == ""
}

// Response represents an OAuth2 token endpoint response, which carries
// either a successful token response or an error response.
// It implements the client.Unmarshaler interface for handling HTTP
// responses in a standardized way.
type Response struct {
	// TokenResponse holds the fields of a successful response.
	TokenResponse TokenResponse
	// ErrorResponse holds the fields of an error response. Check
	// ErrorResponse.IsEmpty to know whether the server returned one.
	ErrorResponse ErrorResponse
}

// Name returns the identifier for this response type.
// It implements the client.Unmarshaler interface.
func (_ Response) Name() string {
	return ResponseName
}

// Unmarshal parses the JSON-encoded response body and stores the result
// in the Response struct. It implements the client.Unmarshaler interface.
//
// The token endpoint returns a single flat JSON object whose meaning
// depends on the HTTP status (RFC 6749 §5.1 and §5.2), so both views
// are decoded from the same body.
//
// Return an error if JSON unmarshaling fails, nil otherwise.
func (r *Response) Unmarshal(_ int, _ http.Header, body []byte) error {
	if err := json.Unmarshal(body, &r.TokenResponse); err != nil {
		return err
	}
	return json.Unmarshal(body, &r.ErrorResponse)
}
