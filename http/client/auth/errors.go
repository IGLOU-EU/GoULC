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

import "errors"

var (
	// ErrNoUserID is returned when the user ID is empty
	ErrNoUserID = errors.New("you must provide a user ID")
	// ErrNoPassword is returned when the password is empty
	ErrNoPassword = errors.New("you must provide a password")
	// ErrNoRealm is returned when the realm is empty
	ErrNoRealm = errors.New("you must provide a realm parameter")
	// ErrNoNonce is returned when the nonce is empty
	ErrNoNonce = errors.New("you must provide a nonce parameter")
	// ErrNoURI is returned when the URI is empty
	ErrNoURI = errors.New("you must provide a URI parameter")
	// ErrUnknownAlgorithm is returned when the algorithm is unknown
	ErrUnknownAlgorithm = errors.New("unknown algorithm provided")
)
