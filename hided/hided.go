/*
 * Copyright 2024 Adrien Kara
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

// Package hided provides types and methods to obfuscate or mask sensitive data.
// It can be used to ensure that sensitive information is not exposed in logs
// or any outputs.
//
// Implementations MUST also implement fmt.Formatter, fmt.GoStringer,
// json.Marshaler, and encoding.TextMarshaler to prevent leaks through
// formatting verbs, JSON serialization, and text marshaling.
package hided

import "fmt"

const obfuscated = "***"

// Hider defines types that can be obfuscated.
type Hider interface {
	// String returns the obfuscated string (expected output: "***")
	fmt.Stringer

	// IsEmpty returns true if the underlying value is empty
	IsEmpty() bool

	// HashMD5 returns an MD5 hashed representation for obfuscation comparison.
	// Note: MD5 is used only for obfuscation, not for cryptographic security.
	HashMD5() string

	// Value returns the underlying value
	Value() any
}

// Value returns the underlying value of h asserted to T. If h is nil or the
// underlying value is not of type T, the zero value of T is returned.
//
// It is a type-safe alternative to a raw type assertion on Hider.Value(),
// avoiding both the panic of a single-return assertion and the boilerplate
// of the comma-ok idiom on every call site.
//
// The zero value on a type mismatch is silent by design: this helper trades
// error reporting for call-site brevity. On paths where an empty secret must
// not pass for a valid one (authentication, credentials), check IsEmpty() on
// the Hider or the result before using it.
func Value[T any](h Hider) T {
	var empty T
	if h == nil {
		return empty
	}

	if v, ok := h.Value().(T); ok {
		return v
	}

	return empty
}
