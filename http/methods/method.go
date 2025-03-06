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

// Package methods provides HTTP method constants as defined in RFC 9110 and RFC 5789.
package methods

// Method represents an HTTP request method.
type Method string

const (
	// GET requests transfer of a current selected representation for the target resource.
	// https://www.rfc-editor.org/rfc/rfc9110#GET
	GET Method = "GET"

	// HEAD is identical to GET except that the server MUST NOT send content in the response.
	// https://www.rfc-editor.org/rfc/rfc9110#HEAD
	HEAD Method = "HEAD"

	// POST requests that the target resource process the representation enclosed in the request.
	// https://www.rfc-editor.org/rfc/rfc9110#POST
	POST Method = "POST"

	// PUT requests that the target resource create or update its state with the provided representation.
	// https://www.rfc-editor.org/rfc/rfc9110#PUT
	PUT Method = "PUT"

	// DELETE requests that the target resource delete its state.
	// https://www.rfc-editor.org/rfc/rfc9110#DELETE
	DELETE Method = "DELETE"

	// CONNECT establishes a tunnel to the server identified by the target resource.
	// https://www.rfc-editor.org/rfc/rfc9110#CONNECT
	CONNECT Method = "CONNECT"

	// OPTIONS requests information about the communication options available for the target resource.
	// https://www.rfc-editor.org/rfc/rfc9110#OPTIONS
	OPTIONS Method = "OPTIONS"

	// TRACE performs a message loop-back test along the path to the target resource.
	// https://www.rfc-editor.org/rfc/rfc9110#TRACE
	TRACE Method = "TRACE"

	// PATCH requests that a set of changes described in the request entity be applied to the target resource.
	// https://www.rfc-editor.org/rfc/rfc5789#section-2
	PATCH Method = "PATCH"
)

var methods = [...]Method{
	GET,
	HEAD,
	POST,
	PUT,
	DELETE,
	CONNECT,
	OPTIONS,
	TRACE,
	PATCH,
}

// IsValid returns true if the method is supported
func (m Method) IsValid() bool {
	for _, method := range methods {
		if method == m {
			return true
		}
	}
	return false
}
