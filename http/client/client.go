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
	"sync"
	"sync/atomic"
	"time"

	"gitlab.com/iglou.eu/goulc/http/client/auth"
	"gitlab.com/iglou.eu/goulc/http/utils"
)

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

	// ErrNilContext is returned when a nil context is provided
	ErrNilContext = errors.New("nil context was provided")

	// ErrNoTrace is returned when a nil trace is provided
	ErrNoTrace = errors.New("nil trace was provided")

	// ErrClientClosed is returned when the client is closed
	ErrClientClosed = errors.New("http client is closed")

	// ErrEmptyMethod is returned when the HTTP method is empty
	ErrEmptyMethod = errors.New("request method cannot be empty")
)

// OptDefault defines secure default options for the client
// - HTTPS only for security
// - Limited redirects (2) to prevent loops
// - No auth forwarding to other hosts
// - 35s timeout to prevent hanging
// - TLS verification enabled
var OptDefault = Options{
	OnlyHTTPS:        true,
	Follow:           true,
	FollowAuth:       false,
	FollowReferer:    true,
	MaxRedirect:      2,
	Timeout:          35 * time.Second,
	DisableTLSVerify: false,
}

// New creates and initializes a new Client with the specified configuration.
// It sets up a client for making HTTP requests or creating child clients
// that inherit its configuration.
//
// The `serverURL` parameter must include the scheme and path.
// The `authenticator` parameter can be nil if no authentication is required.
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
func New(ctx context.Context, serverURL string, authenticator auth.Authenticator, opt *Options, logger *slog.Logger) (Client, error) {
	var err error

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
			errors.New("failed to parse URL "+serverURL), err)
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
	} else {
		opt = &OptDefault
	}

	// Initialize the new client
	main := Client{
		Mu:      &sync.RWMutex{},
		logger:  logger,
		Options: *opt,
		Header:  make(http.Header),
		Query:   make(url.Values),
	}

	main.URL = *parsedURL
	main.URL.Path = utils.PathFormatting(main.URL.Path)
	main.context, main.cancel = context.WithCancel(ctx)

	if authenticator != nil {
		main.Auth = authenticator
	}

	if main.Query, err = url.ParseQuery(main.URL.RawQuery); err != nil {
		return Client{}, errors.Join(ErrInvalidQuery,
			errors.New("failed to parse query "+main.URL.RawQuery), err)
	}

	if main.Options.OnlyHTTPS && parsedURL.Scheme == "http" {
		main.logger.Debug("Scheme updated to HTTPS due to OnlyHTTPS option")
		parsedURL.Scheme = "https"
	}

	return main, nil
}

// NewChild creates a new Client that inherits the parent's configuration
// but operates independently. The new client is isolated from the parent,
// allowing for concurrent modifications without affecting the parent client.
//
// The path parameter is appended to the parent's URL path. If empty,
// the parent's path remains unchanged. The path is automatically formatted
// to ensure proper URL structure.
//
// Example:
//
// parent := client.New("https://api.example.com", nil, nil, nil)
// child := parent.NewChild("/v1/users")
// // child URL will be https://api.example.com/v1/users
func (c *Client) NewChild(path string) *Client {
	child := c.Clone()

	if path != "" {
		newPath := utils.PathFormatting(path)

		if child.URL.Path == "/" {
			child.URL.Path = newPath
		} else {
			child.URL.Path = child.URL.Path + newPath
		}
	}

	c.logger.Debug("new child client created",
		"parent_url", c.URL.String(),
		"child_url", child.URL.String())
	return child
}

// Clone creates and returns a new Client that is a copy of the original.
// The cloned Client shares the same logger and RateLimiter as the original but
// has its own mutex, context, headers, parameters, and error history.
// If the original Client is closed, the cloned Client is also marked as
// closed. The new Client’s context is derived from the original’s context,
// and authentication is cloned if it exists.
func (c *Client) Clone() *Client {
	c.Mu.RLock()

	if c.closed {
		c.Mu.RUnlock()
		return nil
	}

	clone := &Client{
		closed:         c.closed,
		activeRequests: 0,
		logger:         c.logger, // keep original pointer
		closer:         []func() error{},

		Mu: &sync.RWMutex{},
		Options: Options{
			OnlyHTTPS:        c.Options.OnlyHTTPS,
			Follow:           c.Options.Follow,
			FollowAuth:       c.Options.FollowAuth,
			FollowReferer:    c.Options.FollowReferer,
			MaxRedirect:      c.Options.MaxRedirect,
			Timeout:          c.Options.Timeout,
			DisableTLSVerify: c.Options.DisableTLSVerify,
			RateLimiter:      c.Options.RateLimiter, // keep original pointer
		},
		Header:       c.Header.Clone(),
		URL:          c.URL,
		Query:        maps.Clone(c.Query),
		ErrorHistory: []ErrorHistory{},
	}

	clone.context, clone.cancel = context.WithCancel(c.context)

	if c.URL.User != nil {
		user := *c.URL.User
		clone.URL.User = &user
	}

	if c.Auth != nil {
		clone.Auth = c.Auth.Clone()
	}

	c.Mu.RUnlock()

	// Add the new client to the closer list of the parent
	// This ensure that the child client is closed when the parent is closed
	c.Mu.Lock()
	c.closer = append(c.closer, clone.Close)
	c.Mu.Unlock()

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
	c.logger.Debug("flushing headers", "current_headers", slices.Sorted(maps.Keys(c.Header)))

	c.Mu.Lock()
	c.Header = http.Header{}
	c.Mu.Unlock()

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
	c.logger.Debug("flushing query parameters", "current_query", c.Query)

	c.Mu.Lock()
	c.Query = url.Values{}
	c.Mu.Unlock()

	return c
}

// calculateErrorRate determines the error rate for the specified status code,
// removing entries older than one minute to ensure accuracy.
func (c *Client) calculateErrorRate(statusCode int) float64 {
	c.Mu.Lock()
	defer c.Mu.Unlock()

	// Clean old entries (older than 1 minute)
	now := time.Now()
	minTime := now.Add(-time.Minute)
	var newHistory []ErrorHistory
	for _, entry := range c.ErrorHistory {
		if entry.Timestamp.After(minTime) {
			newHistory = append(newHistory, entry)
		}
	}

	// Add current request
	newHistory = append(newHistory, ErrorHistory{
		URL:        c.URL.String(),
		StatusCode: statusCode,
		Timestamp:  now,
		IsError:    statusCode >= 400,
	})
	c.ErrorHistory = newHistory

	// Calculate error rate
	if len(newHistory) == 0 {
		return 0
	}

	var errorCount int
	for _, entry := range newHistory {
		if entry.IsError {
			errorCount++
		}
	}

	return float64(errorCount) / float64(len(newHistory)) * 100
}

// FollowRedirects returns a RedirectFunc that follows HTTP redirects according
// to the client's options. It also records the redirects in the trace
// parameter.
func (c *Client) FollowRedirects(trace *[]Redirects) func(req *http.Request, via []*http.Request) error {
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

		// Apply rate limiting to redirect requests if configured
		if c.Options.RateLimiter != nil {
			if err := c.Options.RateLimiter.Wait(c.context); err != nil {
				return err
			}
		}

		c.logger.Debug("follow redirection",
			"from", prevURL, "to", req.URL.String(),
			"redirect_count", nb, "max_redirect", c.Options.MaxRedirect)
		return nil
	}
}

// Close gracefully shuts down the Client and releases all associated resources.
// It marks the Client as closed to prevent new requests, logs the closure
// process, and waits for active requests to complete within the configured
// timeout. If active requests do not finish before the timeout, a warning is
// logged. After calling Close, the Client cannot be reused.
func (c *Client) Close() error {
	// Lock temporarily to avoid hanging active requests
	c.Mu.Lock()
	if c.closed {
		// In case the context was not closed
		if c.cancel != nil {
			c.cancel()
		}

		c.Mu.Unlock()
		return nil
	}
	c.closed = true
	timeOut := c.Options.Timeout
	c.Mu.Unlock()

	// Log closing
	c.logger.Debug("closing http client",
		"url", c.URL.String(),
		"active_requests", atomic.LoadInt32(&c.activeRequests))

	// Wait for active requests to complete (with timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeOut)
	defer cancel()

	done := make(chan struct{})
	go func() {
		for atomic.LoadInt32(&c.activeRequests) > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		close(done)
	}()

	// Wait for either completion or timeout
	select {
	case <-done:
		c.logger.Debug("http client closed successfully")
	case <-ctx.Done():
		c.logger.Warn("http client close timed out with active requests",
			"active_requests", atomic.LoadInt32(&c.activeRequests),
			"ctx_err", ctx.Err())
	}

	// Clean up resources
	c.Mu.Lock()
	defer c.Mu.Unlock()

	if c.cancel != nil {
		c.cancel()
	}

	c.Options = Options{}
	c.Header = nil
	c.Auth = nil
	c.URL = url.URL{}
	c.Query = nil
	c.ErrorHistory = nil

	// Close all child clients
	wg := sync.WaitGroup{}
	for _, closer := range c.closer {
		wg.Add(1)
		go func(closer func() error) {
			defer wg.Done()
			if closer == nil {
				return
			}

			if err := closer(); err != nil {
				c.logger.Error("error closing child client", "error", err)
			}
		}(closer)
	}
	wg.Wait()

	c.closer = nil
	c.logger = nil

	return nil
}

// DoWithMarshal is a convenience function that performs a client.Do() call but
// with a body Marshaller instance. For nil body, prefer to use Do instead.
func (main *Client) DoWithMarshal(
	method string, body Marshaler, uml Unmarshaler,
) (*Response, error) {
	// Check if client is closed
	if main.IsClosed() {
		return nil, ErrClientClosed
	}

	// Create a copy of the client to avoid modifying the original
	// and potential race conditions
	c := main.Clone() // Clone are thread-safe

	if body == nil {
		return c.Do(method, nil, uml)
	}

	c.logger.Debug("http client marshalling body",
		"marshaller", body.Name(),
		"content_type", body.ContentType())

	bodyData, err := body.Marshal()
	if err != nil {
		return nil, err
	}

	c.Header.Set("Content-Type", body.ContentType())

	return c.Do(method, bodyData, uml)
}

// Do performs an HTTP request with the specified method and body. It manages
// authentication, redirects, and TLS configuration based on the client's
// options. This method is thread-safe and can be invoked concurrently from
// multiple goroutines.
//
// Example:
//
//	resp, err := client.Do(http.MethodGet, nil, &MyResponseType{})
//	if err != nil {
//	    return err
//	}
//	// Use type assertion like resp.BodyUml.(*MyResponseType) to access parsed data
//
// Parameters:
//   - method: The HTTP method to use for the request.
//   - body: The request payload as a byte slice. Can be nil.
//   - uml: An Unmarshaler instance to parse the response body.
//
// Returns:
//   - A pointer to a Response containing the HTTP response details.
//   - An error if the request fails or the client is closed.
func (main *Client) Do(
	method string, body []byte, uml Unmarshaler,
) (*Response, error) {
	// Check if client is closed
	if main.IsClosed() {
		return nil, ErrClientClosed
	}

	// Create a copy of the client to avoid modifying the original
	// and potential race conditions
	c := main.Clone() // Clone are thread-safe
	defer func() {
		c.Close()
		c = nil
	}() // Release resources when done

	// Increment main active requests counter
	atomic.AddInt32(&main.activeRequests, 1)
	defer atomic.AddInt32(&main.activeRequests, -1)

	// Validate input parameters
	if method == "" {
		return nil, errors.Join(ErrInvalidMethod, ErrEmptyMethod)
	}

	// Add query to URL
	if len(c.Query) > 0 {
		c.logger.Debug("encoding query parameters", "query", c.Query)
		c.URL.RawQuery = c.Query.Encode()
	}

	// Create request with context for cancellation/timeout support
	var err error
	var req *http.Request

	if body != nil {
		req, err = http.NewRequestWithContext(c.context,
			string(method), c.URL.String(), bytes.NewReader(body))
	} else {
		req, err = http.NewRequestWithContext(c.context,
			string(method), c.URL.String(), nil)
	}

	if err != nil {
		return nil, err
	}

	// Copy all headers from client to request
	// Using maps.Copy ensures a proper deep copy of the headers
	maps.Copy(req.Header, c.Header)

	if body != nil && req.Header.Get("Content-Type") == "" {
		c.logger.Debug("setting the default content type",
			"content_type", "application/json",
			"body_size", len(body))
		req.Header.Set("Content-Type", "application/json")
	}

	// Handle authentication if configured
	// Some auth methods might need to read the body to generate the auth header
	// (e.g., for signing the request)
	if c.Auth != nil {
		c.logger.Debug("adding authentication header",
			"auth_name", c.Auth.Name())
		name, value, err := c.Auth.Header(method, req.URL, body)
		if err != nil {
			return nil, err
		}
		req.Header.Set(name, value)
	}

	// Initialize redirects tracking
	redirectsVia := make([]Redirects, 0, 1)

	// Create HTTP client with configured timeout and redirect
	client := &http.Client{
		Timeout:       c.Options.Timeout,
		CheckRedirect: c.FollowRedirects(&redirectsVia),
	}

	// Configure TLS if needed
	if c.Options.DisableTLSVerify {
		c.logger.Debug("TLS verification disabled",
			"warning", "insecure connection",
			"host", c.URL.Hostname(),
			"proto", "http/1.1")
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
				ServerName:         c.URL.Hostname(),
				NextProtos:         []string{"http/1.1"},
			},
		}
	}

	c.logger.Debug("executing HTTP request",
		"method", req.Method,
		"url", req.URL.String(),
		"headers", slices.Sorted(maps.Keys(req.Header)))

	start := time.Now()

	// Apply rate limiting to request
	if c.Options.RateLimiter != nil {
		if err := c.Options.RateLimiter.Wait(c.context); err != nil {
			return nil, err
		}
	}

	httpRes, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer httpRes.Body.Close()

	// Create response object with essential info
	res := &Response{
		Success:      httpRes.StatusCode < 400,
		StatusCode:   httpRes.StatusCode,
		Status:       httpRes.Status,
		Proto:        httpRes.Proto,
		Header:       httpRes.Header.Clone(),
		Request:      httpRes.Request,
		raw:          httpRes,
		ResponseTime: time.Since(start),
		Trace:        redirectsVia,
		ErrorRate:    c.calculateErrorRate(httpRes.StatusCode),
	}

	c.logger.Debug("HTTP request",
		"success", res.Success,
		"method", req.Method,
		"path", req.URL.Path,
		"status", res.Status,
		"trace", res.Trace,
		"response_time", res.ResponseTime,
		"error_rate", res.ErrorRate)

	if httpRes.ContentLength == 0 {
		c.logger.Debug("empty response body received")
		return res, nil
	}
	c.logger.Debug("reading response body",
		"status_code", res.StatusCode,
		"content_length", httpRes.ContentLength)

	res.Body, err = io.ReadAll(httpRes.Body)
	if err != nil {
		return nil, errors.Join(ErrRequestFailed, err)
	}

	// Unmarshal response body if an unmarshaler is provided
	// This allows automatic parsing of JSON/XML/etc into structs
	// The unmarshaler has access to both the status code and body
	// to handle different response formats based on status
	if uml != nil {
		c.logger.Debug("unmarshaling response body",
			"unmarshaler", uml.Name(),
			"body_size", len(res.Body))

		res.BodyUml = uml
		if err := res.BodyUml.Unmarshal(
			res.StatusCode, res.Header, res.Body,
		); err != nil {
			return nil, errors.Join(ErrRequestFailed, err)
		}
	}

	return res, nil
}

// IsClosed checks if the client is closed.
// Call Close() if the context is closed but not the client,
// or if the client is closed but not the context.
func (c *Client) IsClosed() bool {
	c.Mu.RLock()

	if c.context.Err() != nil {
		c.Mu.RUnlock()

		if !c.closed {
			c.Close() // Ctx closed but Client open
		}

		return true
	}

	if c.closed {
		c.Mu.RUnlock()

		c.Close() // Client closed but ctx open
		return true
	}

	c.Mu.RUnlock()
	return false
}
