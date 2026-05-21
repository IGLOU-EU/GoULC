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

package main

import (
	"fmt"

	"gitlab.com/iglou.eu/goulc/wildcard"
)

func main() {
	// 1. Match — fastest path, byte by byte, no allocation
	str := "daaadabadmanda"
	pattern := "?a*da*d.?*"
	fmt.Printf("Match(%q, %q) = %v\n", pattern, str, wildcard.Match(pattern, str))

	// 2. MatchFromByte — same byte semantics for []byte inputs
	fmt.Printf("MatchFromByte = %v\n",
		wildcard.MatchFromByte([]byte(pattern), []byte(str)))

	// 3. MatchByRune — slower, but operators apply to whole code points.
	// Multi-byte UTF-8 (here, the emoji) is matched as a single rune.
	emojiStr := "Hello 🌍 World"
	emojiPattern := "Hello ? World"
	fmt.Printf("MatchByRune(%q, %q) = %v\n",
		emojiPattern, emojiStr,
		wildcard.MatchByRune(emojiPattern, emojiStr))

	// With Match (byte-wise), the same pattern fails: '?' matches a single
	// byte and the emoji is 4 bytes long.
	fmt.Printf("Match(%q, %q)      = %v\n",
		emojiPattern, emojiStr,
		wildcard.Match(emojiPattern, emojiStr))

	// 4. Operators recap
	fmt.Println()
	fmt.Println("Operators:")
	fmt.Printf("  '*' zero or more — Match(%q, %q) = %v\n",
		"a*z", "abcz", wildcard.Match("a*z", "abcz"))
	fmt.Printf("  '?' zero or one  — Match(%q, %q) = %v, Match(%q, %q) = %v\n",
		"?at", "cat", wildcard.Match("?at", "cat"),
		"?at", "at", wildcard.Match("?at", "at"))
	fmt.Printf("  '.' exactly one  — Match(%q, %q) = %v\n",
		"f.o", "foo", wildcard.Match("f.o", "foo"))
}
