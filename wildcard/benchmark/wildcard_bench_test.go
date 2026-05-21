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

package wildcard_bench

import (
	"fmt"
	"path/filepath"
	"regexp"
	"testing"

	"gitlab.com/iglou.eu/goulc/wildcard"
)

var TestSet = []struct {
	pattern string
	input   string
}{
	{"", "These aren't the wildcard you're looking for"},
	{"These aren't the wildcard you're looking for", ""},
	{"*", "These aren't the wildcard you're looking for"},
	{"These aren't the wildcard you're looking for", "These aren't the wildcard you're looking for"},
	{"Th.e * the wildcard you?re looking fo?", "These aren't the wildcard you're looking for"},
	{"*🤷🏾‍♂️*", "T🥵🤷🏾‍♂️🥓"},
}

func BenchmarkRegex(b *testing.B) {
	for i, t := range TestSet {
		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				regexp.MatchString(t.pattern, t.input)
			}
		})
	}
}

// BenchmarkRegexPrepared measures matching against a regexp compiled once,
// ahead of the loop. Unlike BenchmarkRegex — which recompiles the pattern on
// every call — this isolates the cost of regexp.Regexp.MatchString alone.
//
// WARNING: a prepared regex is NOT a fair single-shot comparison. It only pays
// off when the same pattern is reused many times, because the compilation cost
// (often the dominant term) is amortised away here. Read this as a best case
// for regexp, not as an apples-to-apples comparison with wildcard.Match, which
// does its full work on every call.
func BenchmarkRegexPrepared(b *testing.B) {
	for i, t := range TestSet {
		b.Run(fmt.Sprint(i), func(b *testing.B) {
			re, err := regexp.Compile(t.pattern)
			if err != nil {
				b.Skipf("pattern is not a valid regexp: %v", err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				re.MatchString(t.input)
			}
		})
	}
}

func BenchmarkFilepath(b *testing.B) {
	for i, t := range TestSet {
		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				filepath.Match(t.pattern, t.input)
			}
		})
	}
}

func BenchmarkOldMatchSimple(b *testing.B) {
	for i, t := range TestSet {
		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				Old_MatchSimple(t.pattern, t.input)
			}
		})
	}
}

func BenchmarkOldMatch(b *testing.B) {
	for i, t := range TestSet {
		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				Old_Match(t.pattern, t.input)
			}
		})
	}
}

func BenchmarkMatch(b *testing.B) {
	for i, t := range TestSet {
		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				wildcard.Match(t.pattern, t.input)
			}
		})
	}
}

func BenchmarkMatchByRune(b *testing.B) {
	for i, t := range TestSet {
		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				wildcard.MatchByRune(t.pattern, t.input)
			}
		})
	}
}

func BenchmarkMatchFromByte(b *testing.B) {
	for i, t := range TestSet {
		pattern := []byte(t.pattern)
		input := []byte(t.input)

		b.Run(fmt.Sprint(i), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				wildcard.MatchFromByte(pattern, input)
			}
		})
	}
}
