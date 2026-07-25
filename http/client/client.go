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

// Package client is designed to be safe for concurrent use and provides
// fluent interfaces. It supports various authentication methods through
// the auth interface and package, automatic handling of redirects,
// customizable TLS settings...
package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"time"

	"gitlab.com/iglou.eu/goulc/http/client/auth"
	"gitlab.com/iglou.eu/goulc/http/path"
)

const LoopRateDuration = 100 * time.Millisecond

var (
	// ErrEmptyServerURL is returned when the server URL is empty
	ErrEmptyServerURL = errors.New("server URL cannot be empty")

	// ErrInvalidURL is returned when the URL cannot be parsed
	ErrInvalidURL = errors.New("invalid server URL cannot be parsed")

	// ErrInvalidQuery is returned when the URL query parameters are invalid
	ErrInvalidQuery = errors.New("invalid URL query parameters")

	// ErrRequestFailed is returned when the request fails
	ErrRequestFailed = errors.New("an error occurred while making the request")

	// ErrTooManyRedirects is returned when the maximum number of redirects
	// is exceeded
	ErrTooManyRedirects = errors.New("too many redirects")

	// ErrInvalidMethod is returned when the HTTP method is invalid
	ErrInvalidMethod = errors.New("invalid HTTP method")

	// ErrInvalidTimeout is returned when the timeout value is invalid
	ErrInvalidTimeout = errors.New("invalid timeout value")

	// ErrInvalidRedirectLimit is returned when the redirect limit is invalid
	ErrInvalidRedirectLimit = errors.New("invalid redirect limit")

	// ErrInvalidBodyLimit is returned when the response body size limit
	// is negative
	ErrInvalidBodyLimit = errors.New("invalid response body size limit")

	// ErrBodyTooLarge is returned when a response body exceeds
	// Options.MaxBodySize
	ErrBodyTooLarge = errors.New(
		"response body exceeds the configured size limit")

	// ErrNilContext is returned when a nil context is provided
	ErrNilContext = errors.New("nil context was provided")

	// ErrNoTrace is returned when a nil trace is provided
	ErrNoTrace = errors.New("nil trace was provided")

	// ErrClientClosed is returned when the client is closed
	ErrClientClosed = errors.New("http client is closed")

	// ErrEmptyMethod is returned when the HTTP method is empty
	ErrEmptyMethod = errors.New("request method cannot be empty")

	// ErrNoResult is returned by Result when the response carries no
	// unmarshaled value of the requested type
	ErrNoResult = errors.New(
		"response has no unmarshaled result of the requested type")
)

// OptDefault defines secure default options for the client
// - HTTPS only for security
// - Limited redirects (2) to prevent loops
// - No auth forwarding to other hosts
// - 35s timeout to prevent hanging
// - TLS verification enabled
// - Response bodies capped at 32 MiB to prevent memory exhaustion
var OptDefault = Options{
	OnlyHTTPS:        true,
	Follow:           true,
	FollowAuth:       false,
	FollowReferer:    true,
	MaxRedirect:      2,
	Timeout:          35 * time.Second,
	DisableTLSVerify: false,
	MaxBodySize:      32 << 20,
}

// New creates and initializes a new Client with the specified configuration.
// It sets up a client for making HTTP requests or creating child clients
// that inherit its configuration.
//
// The `serverURL` parameter must include the scheme and path.
// The `authenticator` parameter can be nil if no authentication is
// required. It is shared between all in-flight requests, so it must be
// safe for concurrent use.
// The `opt` parameter allows customization of client behavior through
// the `Options` struct. The `logger` parameter specifies a custom logger;
// if nil, the default logger is used.
//
// The `ctx` parameter specifies a context for cancellation. If nil,
// `context.Background()` is used.
//
// New validates the `serverURL` and the provided options, ensuring that
// timeout and redirect limits are non-negative and that the context
// is not nil. It removes trailing slashes from the `serverURL` for
// consistency, enforces HTTPS if the `OnlyHTTPS` option is set, formats
// the URL path, and parses query parameters. If an authenticator is provided,
// it is cloned for the new Client.
//
// It returns a Client instance configured with the provided parameters or
// an error if the `serverURL` is invalid or cannot be parsed.
func New(
	ctx context.Context, serverURL string, authenticator auth.Authenticator,
	opt *Options, logger *slog.Logger,
) (Client, error) {
	// Empty url are not allowed
	if serverURL == "" {
		return Client{}, ErrEmptyServerURL
	}

	// Remove trailing slash from URL for consistency
	// This ensures uniform URL handling regardless of input
	if serverURL[len(serverURL)-1] == '/' {
		serverURL = serverURL[:len(serverURL)-1]
	}

	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return Client{}, errors.Join(ErrInvalidURL,
			errors.New("parse URL "+serverURL), err)
	}

	// Set default logger and context
	if logger == nil {
		logger = slog.Default()
	}

	if ctx == nil {
		ctx = context.Background()
	}

	// Validate input parameters
	if opt != nil {
		// Validate timeout
		if opt.Timeout < 0 {
			return Client{}, errors.Join(ErrInvalidTimeout,
				errors.New("timeout must be >= 0, got "+
					strconv.Itoa(int(opt.Timeout.Seconds()))))
		}

		// Validate redirect limit
		if opt.MaxRedirect < 0 {
			return Client{}, errors.Join(ErrInvalidRedirectLimit,
				errors.New("redirect limit must be >= 0, got "+
					strconv.Itoa(opt.MaxRedirect)))
		}

		// Validate response body size limit
		if opt.MaxBodySize < 0 {
			return Client{}, errors.Join(ErrInvalidBodyLimit,
				errors.New("body size limit must be >= 0, got "+
					strconv.FormatInt(opt.MaxBodySize, 10)))
		}
	} else {
		opt = &OptDefault
	}

	query, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		return Client{}, errors.Join(ErrInvalidQuery,
			errors.New("parse query "+parsedURL.RawQuery), err)
	}

	baseURL := *parsedURL
	baseURL.Path = path.Format(baseURL.Path)

	if opt.OnlyHTTPS && baseURL.Scheme == "http" {
		logger.Debug("Scheme updated to HTTPS due to OnlyHTTPS option")
		baseURL.Scheme = "https"
	}

	// Built once and shared with children and request snapshots so the
	// transport connection pool is actually reused across requests.
	// As a consequence, DisableTLSVerify is captured here and cannot be
	// changed after New.
	httpClient := &http.Client{Timeout: opt.Timeout}
	if opt.DisableTLSVerify {
		logger.Debug("TLS verification disabled",
			"warning", "insecure connection",
			"host", baseURL.Hostname(),
			"proto", "http/1.1")
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         baseURL.Hostname(),
				NextProtos:         []string{"http/1.1"},
			},
			// A zero value transport never evicts idle connections
			IdleConnTimeout: 90 * time.Second,
		}
	}

	clientCtx, cancel := context.WithCancel(ctx)

	return Client{
		logger:     logger,
		context:    clientCtx,
		cancel:     cancel,
		httpClient: httpClient,
		Options:    *opt,
		Header:     make(http.Header),
		Auth:       authenticator,
		URL:        baseURL,
		Query:      query,
	}, nil
}

// NewChild creates a new Client that inherits the parent's configuration
// but operates independently. The new client is isolated from the parent,
// allowing for concurrent modifications without affecting the parent client.
// It returns nil if the parent is already closed.
//
// The childPath parameter is appended to the parent's URL path. If empty,
// the parent's path remains unchanged. The path is automatically formatted
// to ensure proper URL structure.
//
// The path is appended as raw, already decoded text: a segment that
// contains "/" or ".." reshapes the resulting URL (".." is resolved by
// the formatting), and "%" is re-encoded when the URL is serialized, so
// pre-escaped input gets double encoded. Use NewChildSegments to inject
// external values safely.
//
// Example:
//
// parent := client.New("https://api.example.com", nil, nil, nil)
// child := parent.NewChild("/v1/users")
// // child URL will be https://api.example.com/v1/users
func (c *Client) NewChild(childPath string) *Client {
	child := c.Clone()
	if child == nil {
		return nil
	}

	if childPath != "" {
		newPath := path.Format(childPath)

		if child.URL.Path == "/" {
			child.URL.Path = newPath
		} else {
			child.URL.Path += newPath
		}
	}

	if c.debugEnabled() {
		c.logger.Debug("new child client created",
			"parent_url", c.URL.String(),
			"child_url", child.URL.String())
	}
	return child
}

// NewChildSegments creates a child Client like NewChild, but treats
// every argument as one literal path segment. Each segment is
// percent-encoded, so values containing "/", "..", "%" or any other
// metacharacter cannot restructure the resulting URL. Empty segments
// are dropped. It returns nil if the parent is already closed.
//
// Use it whenever a path element comes from external input:
//
// child := parent.NewChildSegments("projects", projectID, "tasks", taskID)
func (c *Client) NewChildSegments(segments ...string) *Client {
	child := c.Clone()
	if child == nil || len(segments) == 0 {
		return child
	}

	escaped := make([]string, len(segments))
	for i, segment := range segments {
		escaped[i] = escapeSegment(segment)
	}

	child.URL = *child.URL.JoinPath(escaped...)

	if c.debugEnabled() {
		c.logger.Debug("new child client created",
			"parent_url", c.URL.String(),
			"child_url", child.URL.String())
	}
	return child
}

// escapeSegment keeps a path segment literal once joined. PathEscape
// covers every metacharacter but leaves dots alone, and the URL path
// join would resolve "." and ".." segments away, so those two are
// force-encoded.
func escapeSegment(segment string) string {
	switch segment {
	case ".":
		return "%2E"
	case "..":
		return "%2E%2E"
	default:
		return url.PathEscape(segment)
	}
}

// Clone creates and returns a new Client that is a copy of the original.
// The cloned Client shares the same logger and RateLimiter as the original
// but has its own mutex, context, headers and parameters. If the original
// Client is closed, Clone returns nil. The new Client’s context is derived
// from the original’s context, and authentication is cloned if it exists.
func (c *Client) Clone() *Client {
	// Registration happens under the same lock as the copy so a
	// concurrent Close cannot slip between them and miss the new child
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}

	clone := c.copyLocked()

	// A long-lived clone gets its own authenticator so both lineages
	// evolve independently, unlike per-request snapshots
	if c.Auth != nil {
		clone.Auth = c.Auth.Clone()
	}

	c.closer = append(c.closer, clone.Close)

	return clone
}

// snapshot returns a private copy of the client state so an in-flight
// request is isolated from concurrent mutations. Unlike Clone, the copy
// is not registered in the parent closer list: a per-request registration
// would be retained for the whole parent lifetime. The authenticator is
// shared, not cloned: it owns cross-request state such as token caches,
// which a throwaway copy would silently discard.
func (c *Client) snapshot() *Client {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return nil
	}

	return c.copyLocked()
}

// copyLocked builds the actual copy. The caller must hold c.mu. The
// authenticator is shared per the Authenticator concurrency contract,
// Clone overrides it for long-lived children.
func (c *Client) copyLocked() *Client {
	clone := &Client{
		logger:     c.logger,     // keep original pointer
		httpClient: c.httpClient, // share the pooled transport
		Options:    c.Options,    // shallow copy, RateLimiter is shared
		Header:     c.Header.Clone(),
		Auth:       c.Auth,
		URL:        c.URL,
		Query:      maps.Clone(c.Query),
	}

	clone.context, clone.cancel = context.WithCancel(c.context)

	if c.URL.User != nil {
		user := *c.URL.User
		clone.URL.User = &user
	}

	return clone
}

// FlushHeader safely clears all HTTP headers stored in the Client. This allows
// resetting the headers without creating a new instance.
//
// The method returns the Client to enable method chaining.
//
// Example:
//
// client.FlushHeader().FlushQuery()
func (c *Client) FlushHeader() *Client {
	c.mu.Lock()
	if c.debugEnabled() {
		c.logger.Debug("flushing headers", "current_headers",
			slices.Sorted(maps.Keys(c.Header)))
	}
	c.Header = http.Header{}
	c.mu.Unlock()

	return c
}

// FlushQuery safely clears all URL query parameters stored in the Client.
// This allows resetting the query parameters without creating a new instance.
//
// The method returns the Client to enable method chaining.
//
// Example:
//
// client.FlushQuery().Do(http.MethodGet, nil, nil)
func (c *Client) FlushQuery() *Client {
	c.mu.Lock()
	if c.debugEnabled() {
		c.logger.Debug("flushing query parameters", "current_query", c.Query)
	}
	c.Query = url.Values{}
	c.mu.Unlock()

	return c
}

// FollowRedirects returns a RedirectFunc that follows HTTP redirects according
// to the client's options. It also records the redirects in the trace
// parameter.
func (c *Client) FollowRedirects(
	trace *[]Redirects,
) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		if !c.Options.Follow {
			return http.ErrUseLastResponse
		}

		if trace == nil {
			return ErrNoTrace
		}

		// Remove referer for privacy if configured
		if !c.Options.FollowReferer {
			req.Header.Del("Referer")
		}

		// Remove auth headers when redirecting to different host
		// This prevents credential leakage
		if req.URL.Host != c.URL.Host && !c.Options.FollowAuth {
			req.Header.Del("Authorization")
		}

		// Track this redirect
		var prevURL, prevStatus string
		if len(via) > 1 {
			prev := via[len(via)-1]
			prevURL = prev.URL.String()
			prevStatus = prev.Response.Status
		}
		*trace = append(*trace, Redirects{
			URL:        req.URL.String(),
			Status:     req.Response.Status,
			From:       prevURL,
			FromStatus: prevStatus,
			Timestamp:  time.Now(),
		})

		// Check redirect count to prevent infinite loops
		nb := len(via)
		if nb >= c.Options.MaxRedirect {
			return errors.Join(ErrTooManyRedirects,
				errors.New("stopped after "+strconv.Itoa(nb)+" redirects"))
		}

		// Enforce HTTPS on redirects if configured
		if c.Options.OnlyHTTPS && req.URL.Scheme == "http" {
			req.URL.Scheme = "https"
		}

		// Never let credentials issued over TLS travel on a cleartext
		// downgrade, even toward the same host
		if req.URL.Scheme == "http" && len(via) > 0 &&
			via[len(via)-1].URL.Scheme == "https" {
			req.Header.Del("Authorization")
		}

		// Apply rate limiting to redirect requests if configured
		if c.Options.RateLimiter != nil {
			if err := c.Options.RateLimiter.Wait(c.context); err != nil {
				return err
			}
		}

		if c.debugEnabled() {
			c.logger.Debug("follow redirection",
				"from", prevURL, "to", req.URL.String(),
				"redirect_count", nb, "max_redirect", c.Options.MaxRedirect)
		}
		return nil
	}
}

// Close gracefully shuts down the Client and releases all associated resources.
// It marks the Client as closed to prevent new requests, logs the closure
// process, and waits for active requests to complete within the configured
// timeout. If active requests do not finish before the timeout, a warning is
// logged. Child clients are closed in cascade and their errors are joined
// into the returned error. After calling Close, the Client cannot be reused.
func (c *Client) Close() error {
	// Lock temporarily to avoid hanging active requests
	c.mu.Lock()
	if c.closed {
		// In case the context was not closed
		if c.cancel != nil {
			c.cancel()
		}

		c.mu.Unlock()
		return nil
	}
	c.closed = true
	timeOut := c.Options.Timeout
	c.mu.Unlock()

	// Log closing
	c.logger.Debug("closing http client",
		"url", c.URL.String(),
		"active_requests", c.activeRequests.Load())

	// Wait for active requests to complete (with timeout)
	if c.activeRequests.Load() > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeOut)
		defer cancel()

		done := make(chan struct{})
		go func() {
			defer close(done)

			for c.activeRequests.Load() > 0 {
				select {
				case <-ctx.Done():
					return
				case <-time.After(LoopRateDuration):
				}
			}
		}()

		// Wait for either completion or timeout
		select {
		case <-done:
			c.logger.Debug("http client closed successfully")
		case <-ctx.Done():
			// Own the poller lifetime, it must not outlive Close
			<-done
			c.logger.Warn("http client close timed out with active requests",
				"active_requests", c.activeRequests.Load(),
				"ctx_err", ctx.Err())
		}
	}

	// Clean up resources
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}

	// Drop pooled connections of the client-owned transport so idle
	// sockets and their goroutines do not outlive the client
	if c.httpClient != nil {
		if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
		c.httpClient = nil
	}

	c.Options = Options{}
	c.Header = nil
	c.Auth = nil
	c.URL = url.URL{}
	c.Query = nil

	// Close all child clients, keeping their errors visible to the caller
	var errs []error
	for _, closeChild := range c.closer {
		if closeChild == nil {
			continue
		}

		if err := closeChild(); err != nil {
			errs = append(errs, err)
		}
	}

	c.closer = nil
	c.logger = nil

	return errors.Join(errs...)
}

// DoWithMarshal is a convenience function that performs a client.Do() call but
// with a body Marshaller instance. For nil body, prefer to use Do instead.
func (main *Client) DoWithMarshal(
	method string, body Marshaler, resp Unmarshaler,
) (*Response, error) {
	// Check if client is closed
	if main.IsClosed() {
		return nil, ErrClientClosed
	}

	if body == nil {
		return main.doRequest(method, "", nil, resp)
	}

	bodyData, err := body.Marshal()
	if err != nil {
		return nil, err
	}

	if main.debugEnabled() {
		main.logger.Debug("http client marshalling body",
			"marshaller", body.Name(),
			"content_type", body.ContentType())
	}

	return main.doRequest(method, body.ContentType(), bodyData, resp)
}

// Do performs an HTTP request with the specified method and body. It manages
// authentication, redirects, and TLS configuration based on the client's
// options. This method is thread-safe and can be invoked concurrently from
// multiple goroutines.
//
// Example:
//
//	resp, err := c.Do(http.MethodGet, nil, &MyResponseType{})
//	if err != nil {
//	    return err
//	}
//	// Use client.Result[MyResponseType](resp) to access the typed data
//
// Parameters:
//   - method: The HTTP method to use for the request.
//   - body: The request payload as a byte slice. Can be nil.
//   - respUml: An Unmarshaler instance to parse the response body.
//
// Returns:
//   - A pointer to a Response containing the HTTP response details.
//   - An error if the request fails or the client is closed.
func (main *Client) Do(
	method string, body []byte, respUml Unmarshaler,
) (*Response, error) {
	return main.doRequest(method, "", body, respUml)
}

// doRequest performs the request on a private snapshot of the client so
// concurrent configuration changes cannot race with an in-flight call.
// A non-empty contentType takes precedence over the client-level header.
func (main *Client) doRequest(
	method, contentType string, body []byte, respUml Unmarshaler,
) (*Response, error) {
	// Check if client is closed
	if main.IsClosed() {
		return nil, ErrClientClosed
	}

	// Work on a snapshot so the caller's client stays untouched
	c := main.snapshot()
	if c == nil {
		return nil, ErrClientClosed
	}

	// Releasing the snapshot context is enough, a full Close would
	// spawn a wait cycle per request for no benefit
	defer c.cancel()

	// Increment main active requests counter
	main.activeRequests.Add(1)
	defer main.activeRequests.Add(-1)

	// Validate input parameters
	if method == "" {
		return nil, errors.Join(ErrInvalidMethod, ErrEmptyMethod)
	}

	req, err := c.buildRequest(method, contentType, body)
	if err != nil {
		return nil, err
	}

	// Initialize redirects tracking
	redirectsVia := make([]Redirects, 0, 1)

	// Shallow copy of the shared http.Client: the pooled transport is
	// reused while redirect tracking stays request-scoped
	httpClient := *c.httpClient
	httpClient.Timeout = c.Options.Timeout
	httpClient.CheckRedirect = c.FollowRedirects(&redirectsVia)

	if c.debugEnabled() {
		c.logger.Debug("executing HTTP request",
			"method", req.Method,
			"url", req.URL.String(),
			"headers", slices.Sorted(maps.Keys(req.Header)))
	}

	start := time.Now()

	// Apply rate limiting to request
	if c.Options.RateLimiter != nil {
		if err := c.Options.RateLimiter.Wait(c.context); err != nil {
			return nil, err
		}
	}

	httpRes, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpRes.Body.Close()

	// Create response object with essential info
	resp := &Response{
		Success:      httpRes.StatusCode < http.StatusBadRequest,
		StatusCode:   httpRes.StatusCode,
		Status:       httpRes.Status,
		Proto:        httpRes.Proto,
		Header:       httpRes.Header.Clone(),
		Request:      httpRes.Request,
		ResponseTime: time.Since(start),
		Trace:        redirectsVia,
	}

	if c.debugEnabled() {
		c.logger.Debug("HTTP request",
			"success", resp.Success,
			"method", req.Method,
			"path", req.URL.Path,
			"status", resp.Status,
			"trace", resp.Trace,
			"response_time", resp.ResponseTime)
	}

	if httpRes.ContentLength == 0 {
		c.logger.Debug("empty response body received")
		return resp, nil
	}

	if err := c.readBody(httpRes, resp, respUml); err != nil {
		return nil, err
	}

	return resp, nil
}

// buildRequest assembles the outgoing request from the snapshot state:
// URL query, headers, content type and authentication.
func (c *Client) buildRequest(
	method, contentType string, body []byte,
) (*http.Request, error) {
	debug := c.debugEnabled()

	// Add query to URL
	if len(c.Query) > 0 {
		if debug {
			c.logger.Debug("encoding query parameters", "query", c.Query)
		}
		c.URL.RawQuery = c.Query.Encode()
	}

	// Create request with context for cancellation/timeout support,
	// keeping a nil reader interface when there is no body
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(c.context,
		method, c.URL.String(), bodyReader)
	if err != nil {
		return nil, err
	}

	// Copy all headers from client to request
	// Using maps.Copy ensures a proper deep copy of the headers
	maps.Copy(req.Header, c.Header)

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	if body != nil && req.Header.Get("Content-Type") == "" {
		if debug {
			c.logger.Debug("setting the default content type",
				"content_type", "application/json",
				"body_size", len(body))
		}
		req.Header.Set("Content-Type", "application/json")
	}

	// Handle authentication if configured
	// Some auth methods might need to read the body to generate the auth
	// header (e.g., for signing the request)
	if c.Auth != nil {
		if debug {
			c.logger.Debug("adding authentication header",
				"auth_name", c.Auth.Name())
		}

		if err := c.Auth.Update(); err != nil {
			return nil, err
		}

		name, value, err := c.Auth.Header(method, req.URL, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set(name, value)
	}

	return req, nil
}

// readBody reads the response body under the configured size cap and
// runs the optional unmarshaler.
func (c *Client) readBody(
	httpRes *http.Response, resp *Response, respUml Unmarshaler,
) error {
	debug := c.debugEnabled()

	if debug {
		c.logger.Debug("reading response body",
			"status_code", resp.StatusCode,
			"content_length", httpRes.ContentLength)
	}

	// Cap the read so a hostile server cannot exhaust memory, one
	// extra byte makes an over-limit body distinguishable
	bodyReader := io.Reader(httpRes.Body)
	if limit := c.Options.MaxBodySize; limit > 0 {
		bodyReader = io.LimitReader(httpRes.Body, limit+1)
	}

	var err error
	resp.Body, err = io.ReadAll(bodyReader)
	if err != nil {
		return errors.Join(ErrRequestFailed, err)
	}

	if limit := c.Options.MaxBodySize; limit > 0 &&
		int64(len(resp.Body)) > limit {
		return errors.Join(ErrBodyTooLarge,
			errors.New("limit is "+strconv.FormatInt(limit, 10)+" bytes"))
	}

	// Unmarshal response body if an unmarshaler is provided
	// This allows automatic parsing of JSON/XML/etc into structs
	// The unmarshaler has access to both the status code and body
	// to handle different response formats based on status
	if respUml == nil {
		return nil
	}

	if debug {
		c.logger.Debug("unmarshaling response body",
			"unmarshaler", respUml.Name(),
			"body_size", len(resp.Body))
	}

	resp.BodyUml = respUml
	if err := resp.BodyUml.Unmarshal(
		resp.StatusCode, resp.Header, resp.Body,
	); err != nil {
		return errors.Join(ErrRequestFailed, err)
	}

	return nil
}

// debugEnabled reports whether debug records are collected, letting hot
// paths skip building their log arguments.
func (c *Client) debugEnabled() bool {
	return c.logger.Enabled(c.context, slog.LevelDebug)
}

// IsClosed checks if the client is closed.
// Call Close() if the context is closed but not the client,
// or if the client is closed but not the context.
func (c *Client) IsClosed() bool {
	c.mu.RLock()

	if c.context.Err() != nil {
		c.mu.RUnlock()

		if !c.closed {
			c.Close() // Ctx closed but Client open
		}

		return true
	}

	if c.closed {
		c.mu.RUnlock()

		c.Close() // Client closed but ctx open
		return true
	}

	c.mu.RUnlock()
	return false
}
