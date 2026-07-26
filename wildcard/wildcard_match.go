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

package wildcard

// Pattern operators.
const (
	opStar     = '*'
	opQuestion = '?'
	opDot      = '.'
)

// stackRowSize is the widest pattern (in bytes or runes) whose matching
// row still fits on the stack, so that typical patterns containing '?'
// keep the matchers allocation-free.
const stackRowSize = 64

// The rune matcher below mirrors the generic one line for line: []rune
// cannot join the ~string | ~[]byte constraint because its element type
// differs, so the duplication is the price of a rune-wise variant.

// matchGreedy scans pattern and s with a single '*' save point: on a
// mismatch the scan resumes right after the last '*', one input element
// further. One save point is enough because a later '*' always covers
// what an earlier one could. '?' offers a real zero-or-one choice that
// this scheme cannot represent, so the first '?' reached hands the whole
// job over to matchDP. A '?' never reached cannot change the outcome:
// the greedy scan visits every viable prefix alignment before failing.
func matchGreedy[T ~string | ~[]byte](pattern, s T) bool {
	var pIdx, sIdx int
	starPat, starIn := -1, 0

	for sIdx < len(s) {
		if pIdx < len(pattern) {
			switch pattern[pIdx] {
			case opStar:
				starPat, starIn = pIdx, sIdx
				pIdx++
				continue
			case opQuestion:
				return matchDP(pattern, s)
			case opDot:
				pIdx++
				sIdx++
				continue
			default:
				if pattern[pIdx] == s[sIdx] {
					pIdx++
					sIdx++
					continue
				}
			}
		}

		if starPat == -1 {
			return false
		}
		starIn++
		pIdx = starPat + 1
		sIdx = starIn
	}

	// The exhausted input only accepts operators able to match nothing.
	for pIdx < len(pattern) {
		if pattern[pIdx] != opStar && pattern[pIdx] != opQuestion {
			return false
		}
		pIdx++
	}

	return true
}

// matchDP walks s once while keeping one boolean per pattern prefix,
// telling whether that prefix matches the input consumed so far. Keeping
// every prefix alive at once is what makes '?' a true zero-or-one: both
// readings stay open until the rest of the input settles them, wherever
// the '?' sits and however many there are. Cost is
// O(len(pattern) * len(s)) time and one row of len(pattern)+1 booleans,
// on the stack for patterns shorter than stackRowSize.
func matchDP[T ~string | ~[]byte](pattern, s T) bool {
	var buf [stackRowSize]bool
	row := buf[:]
	if rowLen := len(pattern) + 1; rowLen > stackRowSize {
		row = make([]bool, rowLen)
	} else {
		row = row[:rowLen]
	}

	// Row for the empty input: only prefixes made of operators able to
	// match nothing hold.
	row[0] = true
	for p := 1; p < len(row); p++ {
		op := pattern[p-1]
		row[p] = row[p-1] && (op == opStar || op == opQuestion)
	}

	for i := 0; i < len(s); i++ {
		// diag carries row[p-1] as it was before this input element.
		diag := row[0]
		row[0] = false
		for p := 1; p < len(row); p++ {
			prev := row[p]
			switch pattern[p-1] {
			case opStar:
				row[p] = row[p-1] || prev
			case opQuestion:
				row[p] = row[p-1] || diag
			case opDot:
				row[p] = diag
			default:
				row[p] = diag && pattern[p-1] == s[i]
			}
			diag = prev
		}
	}

	return row[len(pattern)]
}

// matchRunesGreedy is matchGreedy for rune slices.
func matchRunesGreedy(pattern, s []rune) bool {
	var pIdx, sIdx int
	starPat, starIn := -1, 0

	for sIdx < len(s) {
		if pIdx < len(pattern) {
			switch pattern[pIdx] {
			case opStar:
				starPat, starIn = pIdx, sIdx
				pIdx++
				continue
			case opQuestion:
				return matchRunesDP(pattern, s)
			case opDot:
				pIdx++
				sIdx++
				continue
			default:
				if pattern[pIdx] == s[sIdx] {
					pIdx++
					sIdx++
					continue
				}
			}
		}

		if starPat == -1 {
			return false
		}
		starIn++
		pIdx = starPat + 1
		sIdx = starIn
	}

	// The exhausted input only accepts operators able to match nothing.
	for pIdx < len(pattern) {
		if pattern[pIdx] != opStar && pattern[pIdx] != opQuestion {
			return false
		}
		pIdx++
	}

	return true
}

// matchRunesDP is matchDP for rune slices.
func matchRunesDP(pattern, s []rune) bool {
	var buf [stackRowSize]bool
	row := buf[:]
	if rowLen := len(pattern) + 1; rowLen > stackRowSize {
		row = make([]bool, rowLen)
	} else {
		row = row[:rowLen]
	}

	// Row for the empty input: only prefixes made of operators able to
	// match nothing hold.
	row[0] = true
	for p := 1; p < len(row); p++ {
		op := pattern[p-1]
		row[p] = row[p-1] && (op == opStar || op == opQuestion)
	}

	for i := 0; i < len(s); i++ {
		// diag carries row[p-1] as it was before this input element.
		diag := row[0]
		row[0] = false
		for p := 1; p < len(row); p++ {
			prev := row[p]
			switch pattern[p-1] {
			case opStar:
				row[p] = row[p-1] || prev
			case opQuestion:
				row[p] = row[p-1] || diag
			case opDot:
				row[p] = diag
			default:
				row[p] = diag && pattern[p-1] == s[i]
			}
			diag = prev
		}
	}

	return row[len(pattern)]
}
