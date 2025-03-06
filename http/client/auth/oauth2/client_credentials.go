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
	"errors"
	"log/slog"
	"net/url"
	"reflect"
	"strings"
	"time"

	"gitlab.com/iglou.eu/goulc/http/client"
	"gitlab.com/iglou.eu/goulc/http/client/auth"
	"gitlab.com/iglou.eu/goulc/http/methods"
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

// Verify ClientCredentials implements Authenticator interface
var _ auth.Authenticator = &ClientCredentials{}

// ClientCredentials implements the OAuth2 Client Credentials Authentication
// scheme, managing access tokens and handling authentication requests.
type ClientCredentials struct {
	log  *slog.Logger
	http *client.Client

	Config     Config
	ClientAuth ClientCredentialsType

	Token TokenResponse
}

// NewClientCredentials creates a new ClientCredentials instance with
// the specified authentication type, configuration, and logger.
func NewClientCredentials(
	clientAuth ClientCredentialsType, config Config, log *slog.Logger, http *client.Client,
) (*ClientCredentials, error) {
	if log == nil {
		return nil, errors.New("nil logger provided")
	}

	var httpInt *client.Client
	if http != nil {
		httpInt = http
	} else {
		http, err := client.New(
			nil, config.Endpoint.URL,
			nil, &client.OptDefault,
			log.WithGroup("oauth2"))
		if err != nil {
			return nil, err
		}

		httpInt = &http
	}

	return &ClientCredentials{
		log:  log,
		http: httpInt,

		Config:     config,
		ClientAuth: clientAuth,
	}, nil
}

// Name returns the identifier for the Client Credentials authentication method.
func (g *ClientCredentials) Name() string {
	return ClientCredentialsName
}

// Update refreshes the access token if it has expired,
// ensuring valid authentication for requests.
func (g *ClientCredentials) Update() error {
	if g.Token.ExpireAt.After(time.Now()) {
		g.log.Debug("Access token are not expired")
		return nil
	}

	// There is no refresh token on client credentials grant
	// RFC 6749 §4.4.3: https://www.rfc-editor.org/rfc/rfc6749#section-4.4.3
	g.log.Debug("Creation of a new access token waze required",
		"access_token", g.Token)
	return g.newToken()
}

// Header provides the authorization header required
// for authenticated HTTP requests.
func (g *ClientCredentials) Header(
	_ methods.Method, _ *url.URL, _ []byte,
) (string, string, error) {
	return ClientCredentialsHeaderName,
		ClientCredentialsHeaderPrefix + g.Token.Token.Value().(string),
		nil
}

// Clone creates a deep copy of the ClientCredentials instance,
// ensuring thread-safe modifications.
func (g *ClientCredentials) Clone() auth.Authenticator {
	return &ClientCredentials{
		log:  g.log,
		http: g.http.NewChild(""),

		Config:     g.Config,
		ClientAuth: g.ClientAuth,

		Token: g.Token,
	}
}

// newToken requests a new access token from the authorization server
// using client credentials.
func (g *ClientCredentials) newToken() error {
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
		data.Set("client_secret", g.Config.ClientSecret.Value().(string))
	}

	// RFC 6749 §4.4.1: https://www.rfc-editor.org/rfc/rfc6749#section-4.4.1
	c.Header.Set("Authorization", "Basic "+auth.BasicUserPass(
		g.Config.ClientID, g.Config.ClientSecret.Value().(string)))
	// RFC 6749 §4.4.2: https://www.rfc-editor.org/rfc/rfc6749#section-4.4.2
	c.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Due to body presense we need to use a POST type
	// RFC 6749 §3.1: https://www.rfc-editor.org/rfc/rfc6749#section-3.1
	res, err := c.Do(methods.POST, []byte(data.Encode()), &Response{})
	if err != nil {
		return err
	}

	// RFC 6749 §4.4.3: https://www.rfc-editor.org/rfc/rfc6749#section-4.4.3
	if res.StatusCode != 200 {
		g.log.Debug("Unexpected server response",
			"code", res.Status,
			"body", res.Body)
		return errors.New(
			"The authorization server as returned an unexpected status code")
	}

	if len(res.Body) <= 0 {
		return errors.New("The authorization server as returned an empty body")
	}

	// Check the type insert
	tokRes, ok := res.BodyUml.(*Response)
	if !ok {
		g.log.Debug("Type insertion issue",
			"type", reflect.TypeOf(res.BodyUml),
			"content", res.BodyUml)

		return errors.New(
			"The body unmarshaler does not match with the oauth2.Response declaration")
	}

	// Feed the token !
	g.Token = tokRes.TokenResponse
	g.Token.ExpireAt = time.Now().Add(g.Token.ExpiresIn.Duration)

	return nil
}
