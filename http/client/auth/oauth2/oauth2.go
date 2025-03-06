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

// Package oauth2 implements OAuth 2.0 specification.
package oauth2

import (
	"gitlab.com/iglou.eu/goulc/hided"
)

// Config holds the configuration for an OAuth2 client.
// It contains all necessary credentials and endpoints required
// to perform OAuth2 authentication flows.
type Config struct {
	// ClientID is the application's ID as registered with the OAuth2 provider.
	ClientID string

	// ClientSecret is the application's secret as registered with
	// the OAuth2 provider.
	// It is stored as a hided.String for enhanced security.
	ClientSecret hided.String

	// Scopes specifies the permissions being requested from
	// the OAuth2 provider.
	// Each scope represents a distinct access level or permission.
	Scopes []string

	// Endpoint contains the provider-specific OAuth2 endpoint URLs.
	Endpoint Endpoint
}

// Endpoint holds the URLs required for OAuth2 authentication.
// These URLs are provider-specific and are used for different
// parts of the OAuth2 flow.
type Endpoint struct {
	// URL is the base endpoint for the OAuth2 provider.
	URL string

	// Auth is the authorization endpoint where
	// the client obtains authorization.
	Auth string

	// Refresh is the endpoint used to refresh expired access tokens.
	Refresh string
}
