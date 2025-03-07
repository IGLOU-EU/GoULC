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
	"encoding/base64"
	"net/url"

	"gitlab.com/iglou.eu/goulc/hided"
)

const (
	// BasicName is the identifier for this authentication method
	BasicName = "auth.Basic"
	// BasicSeparator is used to separate components in basic calculation
	BasicSeparator = ":"
	// BasicHeaderName is the HTTP header name for authentication
	BasicHeaderName = "Authorization"
	// BasicValuePrefix is the prefix for the basic authentication value
	BasicValuePrefix = "Basic "
)

// Verify Basic implements Authenticator interface
var _ Authenticator = &Basic{}

// Basic struct implements the Authenticator interface
type Basic struct {
	// UserID is the user ID
	UserID string
	// Password is the password in hidden mode
	Password hided.String
}

// NewBasic creates a new Basic authentication instance with
// the provided credentials. It returns an error if the
// provided credentials are empty.
func NewBasic(userID string, password hided.String) (Basic, error) {
	if userID == "" {
		return Basic{}, ErrNoUserID
	}

	if password.Value() == hided.String("").Value() {
		return Basic{}, ErrNoPassword
	}

	return Basic{
		UserID:   userID,
		Password: password,
	}, nil
}

// Name returns the identifier for this authentication method.
func (b *Basic) Name() string {
	return BasicName
}

// Update implements the Authenticator interface.
// This method is a no-op as there is no state to update.
func (b *Basic) Update() error {
	return nil
}

// Header return the Header name and Header line with prefix and base64 value.
// Basic auth does not require method, url or body to build the header.
// RFC 2617 §2: https://www.rfc-editor.org/rfc/rfc2617#section-2
func (b *Basic) Header(_ string, _ *url.URL, _ []byte,
) (string, string, error) {
	return BasicHeaderName, BasicValuePrefix +
		BasicUserPass(b.UserID, b.Password.Value().(string)), nil
}

// Clone creates a deep copy of the instance.
// This ensures that modifications don't affect the original instance.
func (b *Basic) Clone() Authenticator {
	return &Basic{
		UserID:   b.UserID,
		Password: b.Password,
	}
}

// BasicUserPass return the base64 value of userid and password separated by a
// single colon ":". Like defined into the Basic auth RFC.
// RFC 2617 §2: https://www.rfc-editor.org/rfc/rfc2617#section-2
func BasicUserPass(userid, password string) string {
	return base64.StdEncoding.EncodeToString(
		[]byte(userid + BasicSeparator + password))
}
