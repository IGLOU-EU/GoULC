/*
 * Copyright 2026 Adrien Kara
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

package wildcard

import "testing"

// TestMatch only covers the trivial cases handled inline by the public API
// functions. Each case is run against Match, MatchByRune and MatchFromByte,
// which must all agree. The wildcard matching logic itself is tested in the
// source/ package.
func TestMatch(t *testing.T) {
	cases := []struct {
		name    string
		pattern string
		s       string
		want    bool
	}{
		{"empty pattern and empty string", "", "", true},
		{"empty pattern and non-empty string", "", "something", false},
		{"lone star matches anything", "*", "anything", true},
		{"lone star matches empty string", "*", "", true},
		{"exact equality", "exact", "exact", true},
		{"pattern starting with a star but not matching", "*abc", "xyz", false},
		{"delegates to matcher on match", "ex?ct", "exact", true},
		{"delegates to matcher on mismatch", "ex?ct", "different", false},
	}

	matchers := []struct {
		name string
		fn   func(pattern, s string) bool
	}{
		{"Match", Match},
		{"MatchByRune", MatchByRune},
		{"MatchFromByte", func(pattern, s string) bool {
			return MatchFromByte([]byte(pattern), []byte(s))
		}},
	}

	for _, c := range cases {
		for _, m := range matchers {
			if got := m.fn(c.pattern, c.s); got != c.want {
				t.Errorf("%s: %s(%q, %q) = %v, want %v",
					c.name, m.name, c.pattern, c.s, got, c.want)
			}
		}
	}
}
