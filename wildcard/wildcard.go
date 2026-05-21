/*
 * Copyright 2023-2026 Adrien Kara
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

//go:generate go run cmd/build/build.go

// Package wildcard reports whether a string matches a wildcard pattern.
//
// Three operators are supported:
//   - '*' matches zero or more characters
//   - '?' matches zero or one character
//   - '.' matches exactly one character
//
// Any other character must match itself.
package wildcard

import "bytes"

// Match reports whether s matches pattern, comparing byte by byte and
// without allocating. Against multi-byte UTF-8 the operators apply to
// bytes, not whole characters; use MatchByRune when that matters.
func Match(pattern, s string) bool {
	if pattern == "" {
		return s == pattern
	}
	if pattern == "*" || s == pattern {
		return true
	}

	return matchByString(pattern, s)
}

// MatchByRune reports whether s matches pattern, comparing rune by rune,
// so the operators apply to whole Unicode code points. Converting pattern
// and s to runes allocates; prefer Match when byte semantics are enough.
func MatchByRune(pattern, s string) bool {
	if pattern == "" {
		return s == pattern
	}
	if pattern == "*" || s == pattern {
		return true
	}

	return matchByRunes([]rune(pattern), []rune(s))
}

// MatchFromByte is Match for byte slices: it reports whether s matches
// pattern, with the same byte-wise semantics and without allocation.
func MatchFromByte(pattern, s []byte) bool {
	if len(pattern) == 0 {
		return len(s) == 0
	}
	if string(pattern) == "*" || bytes.Equal(pattern, s) {
		return true
	}

	return matchByByte(pattern, s)
}
