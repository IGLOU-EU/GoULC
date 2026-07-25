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

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// refMatch is the naive recursive oracle for the documented semantics:
// anchored match, '*' zero or more, '?' zero or one, '.' exactly one.
// It stays deliberately naive (worst case exponential) because an
// obviously correct reference beats a fast one in a test, and it only
// runs on small inputs.
func refMatch(pattern, s string) bool {
	if len(pattern) == 0 {
		return len(s) == 0
	}
	switch pattern[0] {
	case '*':
		return refMatch(pattern[1:], s) ||
			(len(s) > 0 && refMatch(pattern, s[1:]))
	case '?':
		return refMatch(pattern[1:], s) ||
			(len(s) > 0 && refMatch(pattern[1:], s[1:]))
	case '.':
		return len(s) > 0 && refMatch(pattern[1:], s[1:])
	default:
		return len(s) > 0 && pattern[0] == s[0] &&
			refMatch(pattern[1:], s[1:])
	}
}

// byteMatchers groups the two byte-wise entry points, which must always
// agree. MatchByRune is added by tests whose cases are pure ASCII, where
// rune-wise and byte-wise semantics coincide.
var byteMatchers = []struct {
	name string
	fn   func(pattern, s string) bool
}{
	{"Match", Match},
	{"MatchFromByte", func(pattern, s string) bool {
		return MatchFromByte([]byte(pattern), []byte(s))
	}},
	{"MatchByRune", MatchByRune},
}

func TestMatch(t *testing.T) {
	// Patterns longer than the internal stack buffer take the heap
	// allocated row, so they deserve their own cases.
	longA := strings.Repeat("a", 70)

	tests := []struct {
		name        string
		givePattern string
		giveInput   string
		want        bool
	}{
		{"empty pattern empty input", "", "", true},
		{"empty pattern non-empty input", "", "a", false},
		{"lone star empty input", "*", "", true},
		{"lone star any input", "*", "anything", true},
		{"lone question empty input", "?", "", true},
		{"lone question one char", "?", "a", true},
		{"lone question two chars", "?", "ab", false},
		{"lone dot empty input", ".", "", false},
		{"lone dot one char", ".", "a", true},
		{"lone dot two chars", ".", "ab", false},

		{"exact equality", "exact", "exact", true},
		{"case sensitive", "exact", "Exact", false},
		{"literal mismatch", "abc", "abd", false},
		{"anchored no suffix slack", "abc", "abcd", false},
		{"anchored no substring search", "bc", "abc", false},

		{"question median zero", "a?b", "ab", true},
		{"question median one", "a?b", "axb", true},
		{"question median too many", "a?b", "axxb", false},
		{"question median repeated chars", "aa?a", "aaa", true},
		{"question positional todo case", "a?cd", "acd", true},
		{"question leading zero", "?at", "at", true},
		{"question leading one", "?at", "cat", true},
		{"question trailing zero", "do?", "do", true},
		{"question trailing one", "do?", "dog", true},
		{"two questions both zero", "??", "", true},
		{"two questions one char", "??", "a", true},
		{"two questions two chars", "??", "ab", true},
		{"two questions three chars", "??", "abc", false},
		{"two interacting questions both one", "?aa?z", "aaaaz", true},
		{"two interacting questions both zero", "?aa?z", "aaz", true},
		{"two interacting questions one of two", "?aa?z", "aaaz", true},
		{"two interacting questions overflow", "?aa?z", "aaaaaz", false},
		{"question unreachable on mismatch", "ab?c", "zz", false},
		{"question then dot", "?.", "a", true},
		{"question then dot empty input", "?.", "", false},
		{"dot then question", ".?", "a", true},
		{"question dot star", "?.*", "a", true},

		{"star zero chars", "a*b", "ab", true},
		{"star many chars", "a*b", "aXYZb", true},
		{"multi star all zero", "a*b*c", "abc", true},
		{"multi star spread", "a*b*c", "aXbYc", true},
		{"consecutive stars", "a**b", "ab", true},
		{"star retry on repeated prefix", "*ab", "aab", true},
		{"star cannot reorder", "*ab", "ba", false},
		{"trailing star zero", "ab*", "ab", true},
		{"leading star no match", "*abc", "xyz", false},
		{"literal after empty input", "a*", "", false},
		{"star around single char", "*a*", "a", true},
		{"star then trailing question", "a*b?", "aXb", true},
		{"trailing star and question", "ab*?", "ab", true},
		{"star then dot empty input", "*.", "", false},
		{"star then dot one char", "*.", "a", true},
		{"star absorbs then question zero", "*?", "abc", true},

		{"dot between literals", "f.o", "foo", true},
		{"two dots", "..", "ab", true},
		{"dot median", "a.c", "abc", true},
		{"dot needs one char", "a.c", "ac", false},
		{"trailing dot empty rest", "ab.", "ab", false},

		{
			"mixed operators matching sentence",
			". big?brown fox jumps over * wildcard. friend??",
			"A big brown fox jumps over the lazy dog," +
				" with all there wildcards friends",
			true,
		},
		{
			"mixed operators failing sentence",
			". big?brown fox jumps over * wildcard. friend??",
			"A big brown fox fails to jump over the lazy dog," +
				" with all there wildcards friends",
			false,
		},

		{"long pattern question zero", longA + "?z", longA + "z", true},
		{"long pattern question one", longA + "?z", longA + "xz", true},
		{"long pattern question overflow", longA + "?z", longA + "xyz", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, m := range byteMatchers {
				got := m.fn(tt.givePattern, tt.giveInput)
				if got != tt.want {
					t.Errorf("%s(%q, %q) = %v, want %v",
						m.name, tt.givePattern, tt.giveInput,
						got, tt.want)
				}
			}
		})
	}
}

// TestMatch_utf8 pins down where the byte-wise and rune-wise semantics
// diverge on multi-byte UTF-8 and on invalid bytes, which MatchByRune
// decodes as U+FFFD.
func TestMatch_utf8(t *testing.T) {
	tests := []struct {
		name        string
		givePattern string
		giveInput   string
		wantByte    bool
		wantRune    bool
	}{
		{"dot vs two-byte rune", ".", "é", false, true},
		{"dot vs four-byte rune", ".", "🌍", false, true},
		{"two dots vs one two-byte rune", "..", "é", true, false},
		{"question inside word", "h?llo", "héllo", false, true},
		{"question median rune", "a?b", "aéb", false, true},
		{"star median rune", "a*b", "aéb", true, true},
		{"emoji equality", "🌍", "🌍", true, true},
		{"star around emoji cluster", "*🤷🏾‍♂️*", "T🥵🤷🏾‍♂️🥓", true, true},
		{"invalid bytes decode to same rune", "\xff", "\xfe", false, true},
		{"identical invalid byte", "\xff", "\xff", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Match(tt.givePattern, tt.giveInput); got != tt.wantByte {
				t.Errorf("Match(%q, %q) = %v, want %v",
					tt.givePattern, tt.giveInput, got, tt.wantByte)
			}

			got := MatchFromByte(
				[]byte(tt.givePattern), []byte(tt.giveInput))
			if got != tt.wantByte {
				t.Errorf("MatchFromByte(%q, %q) = %v, want %v",
					tt.givePattern, tt.giveInput, got, tt.wantByte)
			}

			if got := MatchByRune(tt.givePattern, tt.giveInput); got != tt.wantRune {
				t.Errorf("MatchByRune(%q, %q) = %v, want %v",
					tt.givePattern, tt.giveInput, got, tt.wantRune)
			}
		})
	}
}

// TestMatch_agreesWithReference sweeps every pattern up to length 5 over
// the alphabet {a, b, *, ?, .} against every input up to length 5 over
// {a, b}, and checks all entry points against the recursive oracle. The
// small alphabet is enough to hit every operator interaction, including
// the multi-question backtracking cases.
func TestMatch_agreesWithReference(t *testing.T) {
	patterns := enumerate("ab*?.", 5)
	inputs := enumerate("ab", 5)

	for _, pattern := range patterns {
		for _, input := range inputs {
			want := refMatch(pattern, input)
			for _, m := range byteMatchers {
				if got := m.fn(pattern, input); got != want {
					t.Fatalf("%s(%q, %q) = %v, reference says %v",
						m.name, pattern, input, got, want)
				}
			}
		}
	}
}

// enumerate returns every string of length 0 to maxLen over alphabet.
func enumerate(alphabet string, maxLen int) []string {
	out := []string{""}
	prev := []string{""}
	for l := 1; l <= maxLen; l++ {
		next := make([]string, 0, len(prev)*len(alphabet))
		for _, p := range prev {
			for i := 0; i < len(alphabet); i++ {
				next = append(next, p+string(alphabet[i]))
			}
		}
		out = append(out, next...)
		prev = next
	}
	return out
}

// FuzzMatch checks differential properties rather than a tautology: the
// string and []byte paths must agree everywhere, the rune path must agree
// on pure ASCII, and small cases must agree with the recursive oracle.
func FuzzMatch(f *testing.F) {
	seeds := []struct{ pattern, s string }{
		{"", ""},
		{"a?b", "ab"},
		{"?aa?z", "aaaaz"},
		{"a*b*c", "aXbYc"},
		{"*ab", "aab"},
		{".?*", "xy"},
		{"\xff", "\xfe"},
		{"h?llo", "héllo"},
		{"*🤷🏾‍♂️*", "T🥵🤷🏾‍♂️🥓"},
		{
			"Th.e * the wildcard you?re looking fo?",
			"These aren't the wildcard you're looking for",
		},
	}
	for _, seed := range seeds {
		f.Add(seed.pattern, seed.s)
	}

	f.Fuzz(func(t *testing.T, pattern, s string) {
		got := Match(pattern, s)

		if fromByte := MatchFromByte([]byte(pattern), []byte(s)); fromByte != got {
			t.Errorf("Match(%q, %q) = %v but MatchFromByte = %v",
				pattern, s, got, fromByte)
		}

		// Must not panic on arbitrary bytes, and on pure ASCII the
		// rune-wise semantics collapse onto the byte-wise ones.
		byRune := MatchByRune(pattern, s)
		if isASCII(pattern) && isASCII(s) && byRune != got {
			t.Errorf("Match(%q, %q) = %v but MatchByRune = %v on ASCII",
				pattern, s, got, byRune)
		}

		// The oracle is exponential in the worst case, keep it small.
		if len(pattern) <= 10 && len(s) <= 10 {
			if want := refMatch(pattern, s); got != want {
				t.Errorf("Match(%q, %q) = %v, reference says %v",
					pattern, s, got, want)
			}
		}
	})
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}
