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

package oauth2

import (
	"encoding/json"
	"time"

	"gitlab.com/iglou.eu/goulc/duration"
	"gitlab.com/iglou.eu/goulc/hided"
)

// TokenResponse represents successful access token response
// RFC 6749 §5.1: https://www.rfc-editor.org/rfc/rfc6749#section-5.1
type TokenResponse struct {
	Token        hided.String      `json:"access_token"`
	TokenType    string            `json:"token_type"`
	ExpiresIn    duration.Duration `json:"expires_in"`
	RefreshToken hided.String      `json:"refresh_token"`
	Scope        string            `json:"scope"`

	// Store the issued date
	// RFC 6749 §5.1: https://www.rfc-editor.org/rfc/rfc6749#section-5.1
	expireAt time.Time
}

// ErrorResponse represents error response
// RFC 6749 §5.2: https://www.rfc-editor.org/rfc/rfc6749#section-5.2
type ErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	ErrorURI         string `json:"error_uri"`
}

// Response represents an OAuth2 response that can contain either
// a successful token response or an error response.
// It implements the response.Response interface for handling HTTP responses
// in a standardized way.
type Response struct {
	TokenResponse
	ErrorResponse
}

// Name returns the identifier for this response type.
// It implements the response.Response interface.
func (r Response) Name() string {
	return "oauth2.Response"
}

// Unmarshal parses the JSON-encoded response body and stores the result
// in the Response struct. It implements the response.Response interface.
//
// Return an error if JSON unmarshaling fails, nil otherwise.
func (r *Response) Unmarshal(_ int, body []byte) error {
	return json.Unmarshal(body, r)
}
