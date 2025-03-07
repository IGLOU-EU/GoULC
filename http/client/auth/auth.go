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

// Package auth implements authentication mechanisms for HTTP requests.
// It provides a variety of authentication methods, including Basic
// authentication, to manage and apply authentication headers and handle
// credential updates effectively.
package auth

import (
	"net/url"
)

// Authenticator defines the interface for authentication mechanisms.
type Authenticator interface {
	// Name returns the name of the authenticator.
	Name() string

	// Update refreshes the authenticator's state or credentials.
	Update() error

	// Header generates the authentication header based on the provided method,
	// URL, and body. It returns the header key, header value, and any error
	// encountered.
	Header(method string, url *url.URL, body []byte) (string, string, error)

	// Clone creates and returns a copy of the authenticator.
	Clone() Authenticator
}
