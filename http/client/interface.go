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

package client

import (
	"context"
	"net/http"
)

// Ratelimiter defines an interface for rate limiting HTTP requests.
// Implementations of this interface can be used to control the rate
// at which requests are made to the server.
type Ratelimiter interface {
	// Wait blocks until the rate limit allows another request or the context
	// is canceled. It returns an error if the context is canceled or if the
	// rate limiter encounters an error.
	Wait(ctx context.Context) (err error)
}

// Nameable defines an interface for types that can return their name.
type Nameable interface {
	// Name returns the name of the unmarshaler.
	Name() string
}

// Marshaller defines an interface for types that can marshal themselves
// into a byte slice.
type Marshaller interface {
	Nameable

	// ContentType returns the content type of the marshalled data.
	ContentType() string

	// Marshal serializes the receiver into a byte slice.
	// It returns an error if the marshalling fails.
	Marshal() ([]byte, error)
}

// Unmarshaler defines an interface for types that can unmarshal themselves
// from a byte slice.
type Unmarshaler interface {
	Nameable

	// Unmarshal parses the byte slice and populates the receiver.
	// It takes the HTTP response status code, header response and
	// the byte slice as arguments.
	Unmarshal(statusCode int, header http.Header, body []byte) error
}
