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

// Package wildcard reports whether a string matches a wildcard pattern.
//
// Three operators are supported:
//   - '*' matches zero or more characters
//   - '?' matches zero or one character
//   - '.' matches exactly one character
//
// Any other character must match itself. Matching is anchored: the whole
// input is compared against the whole pattern, there is no substring
// search.
//
// The '?' operator is a true zero-or-one: both readings are explored, so
// "a?b" matches "ab" as well as "axb", wherever the '?' sits.
//
// The worst-case cost is O(len(pattern) * len(s)) and adversarial inputs
// reach it (for instance '*' followed by a long literal tail), so bound
// both lengths before matching when the pattern and the input are both
// untrusted.
package wildcard

import "bytes"

// Match reports whether s matches pattern, comparing byte by byte.
// Against multi-byte UTF-8 the operators apply to bytes, not whole
// characters; use MatchByRune when that matters.
//
// Match never allocates, except for patterns longer than 63 bytes that
// contain '?'.
func Match(pattern, s string) bool {
	if pattern == "" {
		return s == pattern
	}
	if pattern == "*" || s == pattern {
		return true
	}

	return matchGreedy(pattern, s)
}

// MatchByRune reports whether s matches pattern, comparing rune by rune,
// so the operators apply to whole Unicode code points. Converting pattern
// and s to runes allocates; prefer Match when byte semantics are enough.
//
// Invalid UTF-8 bytes are decoded as U+FFFD before matching, so two
// distinct invalid bytes compare as equal: MatchByRune("\xff", "\xfe")
// is true.
func MatchByRune(pattern, s string) bool {
	if pattern == "" {
		return s == pattern
	}
	if pattern == "*" || s == pattern {
		return true
	}

	return matchRunesGreedy([]rune(pattern), []rune(s))
}

// MatchFromByte is Match for byte slices: it reports whether s matches
// pattern, with the same byte-wise semantics and the same allocation
// behavior.
func MatchFromByte(pattern, s []byte) bool {
	if len(pattern) == 0 {
		return len(s) == 0
	}
	if string(pattern) == "*" || bytes.Equal(pattern, s) {
		return true
	}

	return matchGreedy(pattern, s)
}
