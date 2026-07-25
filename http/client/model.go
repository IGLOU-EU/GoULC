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

package client

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/iglou.eu/goulc/http/client/auth"
)

// Redirects stores information about a HTTP redirection.
type Redirects struct {
	URL        string
	Status     string
	From       string
	FromStatus string
	Timestamp  time.Time
}

// Options configures the behavior of the HTTP client.
type Options struct {
	// OnlyHTTPS enforces the use of HTTPS protocol.
	// Default: true
	OnlyHTTPS bool `json:",omitzero"`

	// Follow enables automatic following of HTTP 3xx redirects.
	// Default: true
	Follow bool `json:",omitzero"`

	// FollowAuth determines if authorization headers should be preserved
	// when redirecting to a different host. It's false by default to prevent
	// credential leakage.
	//
	// Its practical scope is limited to subdomains and port changes: the
	// standard library strips the Authorization header on its own, before
	// this policy runs, whenever the redirect target hostname is neither
	// the initial one nor a subdomain of it. That comparison ignores ports.
	// Default: false
	FollowAuth bool `json:",omitzero"`

	// FollowReferer preserves the referer header on redirects.
	// Default: true
	FollowReferer bool `json:",omitzero"`

	// MaxRedirect specifies the maximum number of redirects to follow.
	// Default: 2
	MaxRedirect int `json:",omitzero"`

	// Timeout sets the maximum duration for the entire request.
	// Default: 35s
	Timeout time.Duration `json:",omitzero"`

	// DisableTLSVerify skips TLS certificate validation when true.
	// Default: false
	DisableTLSVerify bool `json:",omitzero"`

	// RateLimiter allows for rate limiting by implementing the Wait method.
	// Default: nil
	RateLimiter Ratelimiter `json:"-"`
}

// Client manages its own configuration. The configuration can be safely
// modified using the provided methods. It also supports creating child clients
// that inherit the parent's configuration but can be modified independently.
type Client struct {
	closed         bool
	activeRequests atomic.Int32
	logger         *slog.Logger

	// mu guards the client state so concurrent use stays safe
	mu sync.RWMutex

	closer  []func() error
	context context.Context
	cancel  context.CancelFunc

	// httpClient is built once by New and shared with children and
	// request snapshots so the transport connection pool is reused
	httpClient *http.Client

	// Options contains the client's configuration settings
	Options Options

	// Header stores HTTP headers to be sent with requests
	Header http.Header

	// Auth contains authentication configuration
	Auth auth.Authenticator

	// URL stores the base URL for requests
	URL url.URL

	// Query stores URL query parameters
	Query url.Values
}

// Response encapsulates the HTTP response details and provides access to
// response data. It includes the status code, headers, body, and performance
// metrics of the response, as well as additional metadata about the request
// and response.
//
// The Response can optionally unmarshal the body data into a structured format
// using an Unmarshaler implementation. This allows for automatic parsing of
// response data into appropriate Go types.
type Response struct {
	// Success indicates if the request was successful
	// (status code < 400, with special handling for 401)
	Success bool

	// StatusCode is the HTTP response status code
	StatusCode int

	// Status is the HTTP status line
	Status string

	// Proto is the HTTP protocol version used in the response
	Proto string

	// Header contains the response headers
	Header http.Header

	// Body contains the raw response body
	Body []byte

	// BodyUml provides interface for the unmarshaling process if any
	BodyUml Unmarshaler

	// Request contains the original HTTP request
	Request *http.Request

	// ResponseTime is the total time taken for the request to complete
	ResponseTime time.Duration

	// Trace contains information about the redirects
	// that occurred during the request
	Trace []Redirects
}
