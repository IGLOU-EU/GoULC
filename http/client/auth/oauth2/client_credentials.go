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
	"context"
	"errors"
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	net_http "net/http"

	"gitlab.com/iglou.eu/goulc/hided"
	"gitlab.com/iglou.eu/goulc/http/client"
	"gitlab.com/iglou.eu/goulc/http/client/auth"
)

var (
	// ErrorUnexpectedStatusCode is returned when the authorization server
	// returns a non-200 status code.
	ErrorUnexpectedStatusCode = errors.New(
		"The authorization server as returned an unexpected status code")
	// ErrorEmptyBody is returned when the authorization server returns an
	// empty body doesn't contain the expected token.
	ErrorEmptyBody = errors.New(
		"The authorization server as returned an empty body")
	// ErrorNoToken is returned when the authorization server returns a
	// response without a token.
	ErrorBodyUnmarshaler = errors.New(
		"The authorization server as returned a response without a token")
)

// ClientCredentialsType defines where client credentials are sent,
// either in the header or in the body.
type ClientCredentialsType uint8

const (
	// ClientCredentialsName is the identifier for
	// Client Credentials authentication method
	ClientCredentialsName = "oauth2.ClientCredentials"
	// ClientCredentialsHeaderName is the HTTP header name for authentication
	ClientCredentialsHeaderName = "Authorization"
	// ClientCredentialsHeaderPrefix is the prefix for the authentication value
	ClientCredentialsHeaderPrefix = "Bearer "

	// ClientInHeader indicates that client credentials are sent in the header.
	ClientInHeader ClientCredentialsType = iota
	// ClientInBody indicates that client credentials are sent in the body.
	ClientInBody
)

// expiryMargin is the anticipation window applied to token expiry: a
// token that close to expiring is refreshed early so it cannot be
// rejected while a request is in flight. Tokens whose lifetime is
// shorter than the margin are refreshed on every request.
const expiryMargin = 30 * time.Second

// Verify ClientCredentials implements Authenticator interface
var _ auth.Authenticator = &ClientCredentials{}

// ClientCredentials implements the OAuth2 Client Credentials Authentication
// scheme, managing access tokens and handling authentication requests.
type ClientCredentials struct {
	log  *slog.Logger
	http *client.Client

	// mu guards Token so a refresh cannot race a concurrent header read
	mu sync.Mutex

	// now returns the current time, injectable so tests can pin expiry
	now func() time.Time

	Config     Config
	ClientAuth ClientCredentialsType

	Token TokenResponse
}

// NewClientCredentials creates a new ClientCredentials instance with
// the specified authentication type, configuration, and logger.
func NewClientCredentials(
	clientAuth ClientCredentialsType, config Config, log *slog.Logger,
	http *client.Client,
) (*ClientCredentials, error) {
	cc := &ClientCredentials{
		log:  log,
		http: http,

		Config:     config,
		ClientAuth: clientAuth,
	}

	if log == nil {
		cc.log = slog.Default()
	}

	if http == nil {
		http, err := client.New(
			context.Background(), config.Endpoint.URL, nil,
			&client.OptDefault, cc.log.WithGroup("oauth2"))
		if err != nil {
			return nil, err
		}

		cc.http = &http
	}

	return cc, nil
}

// Name returns the identifier for the Client Credentials authentication method.
func (_ *ClientCredentials) Name() string {
	return ClientCredentialsName
}

// Update refreshes the access token when it is expired or about to
// expire, ensuring valid authentication for requests. It is safe for
// concurrent use.
func (g *ClientCredentials) Update() error {
	// The whole check-then-refresh sequence stays under the lock so
	// two concurrent calls cannot both decide to refresh
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.Token.ExpireAt.After(g.timeNow().Add(expiryMargin)) {
		g.log.Debug("Access token is still valid")
		return nil
	}

	// There is no refresh token on client credentials grant
	// RFC 6749 §4.4.3: https://www.rfc-editor.org/rfc/rfc6749#section-4.4.3
	g.log.Debug("Requesting a new access token")
	return g.newToken()
}

// Header provides the authorization header required
// for authenticated HTTP requests. It is safe for concurrent use.
func (g *ClientCredentials) Header(_ string, _ *url.URL, _ []byte,
) (headerKey, headerValue string, err error) {
	g.mu.Lock()
	token := hided.Value[string](g.Token.Token)
	g.mu.Unlock()

	return ClientCredentialsHeaderName,
		ClientCredentialsHeaderPrefix + token,
		nil
}

// Clone creates an independent copy of the ClientCredentials instance
// with its own token cache and its own child HTTP client. When the
// underlying HTTP client is already closed, the copy is created without
// one and its next token refresh fails with client.ErrClientClosed.
func (g *ClientCredentials) Clone() auth.Authenticator {
	g.mu.Lock()
	defer g.mu.Unlock()

	var child *client.Client
	if g.http != nil {
		child = g.http.NewChild("")
	}

	return &ClientCredentials{
		log:  g.log,
		http: child,
		now:  g.now,

		Config:     g.Config,
		ClientAuth: g.ClientAuth,

		Token: g.Token,
	}
}

// newToken requests a new access token from the authorization server
// using client credentials.
func (g *ClientCredentials) newToken() error {
	var tokenResp Response

	// New request to Auth
	c := g.http.NewChild(g.Config.Endpoint.Auth)

	// Build the request body
	// RFC 6749 §4.4.2: https://www.rfc-editor.org/rfc/rfc6749#section-4.4.2
	data := url.Values{}
	data.Set("grant_type", "client_credentials")

	// Add scope if specified
	if len(g.Config.Scopes) > 0 {
		data.Set("scope", strings.Join(g.Config.Scopes, " "))
	}

	// Add auth to body if requested
	if g.ClientAuth == ClientInBody {
		data.Set("client_id", g.Config.ClientID)
		data.Set("client_secret", hided.Value[string](g.Config.ClientSecret))
	}

	// RFC 6749 §4.4.1: https://www.rfc-editor.org/rfc/rfc6749#section-4.4.1
	c.Header.Set("Authorization", "Basic "+auth.BasicUserPass(
		g.Config.ClientID, hided.Value[string](g.Config.ClientSecret)))
	// RFC 6749 §4.4.2: https://www.rfc-editor.org/rfc/rfc6749#section-4.4.2
	c.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Due to body presence we need to use a POST type
	// RFC 6749 §3.1: https://www.rfc-editor.org/rfc/rfc6749#section-3.1
	res, err := c.Do(net_http.MethodPost, []byte(data.Encode()), &tokenResp)
	if err != nil {
		return err
	}

	// RFC 6749 §4.4.3: https://www.rfc-editor.org/rfc/rfc6749#section-4.4.3
	if res.StatusCode != net_http.StatusOK {
		g.log.Debug("Unexpected server response",
			"code", res.Status,
			"body", res.Body)
		return ErrorUnexpectedStatusCode
	}

	// Check the body size
	if len(res.Body) == 0 {
		return ErrorEmptyBody
	}

	// Check if the body contains the expected token
	if tokenResp.Token.IsEmpty() {
		g.log.Debug("No token found in the response",
			"unmarshaler", tokenResp,
			"raw", string(res.Body))
		return ErrorBodyUnmarshaler
	}

	// Feed the token !
	// The expires_in value is a lifetime in seconds (RFC 6749 §5.1)
	g.Token = tokenResp.TokenResponse
	g.Token.ExpireAt = g.timeNow().Add(
		time.Duration(g.Token.ExpiresIn) * time.Second)

	return nil
}

// timeNow returns the injected clock when one is set, so instances
// built without the constructor keep working on the real clock.
func (g *ClientCredentials) timeNow() time.Time {
	if g.now != nil {
		return g.now()
	}
	return time.Now()
}
