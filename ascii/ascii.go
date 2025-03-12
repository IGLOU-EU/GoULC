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

// Package ascii implements ASCII string validation functions.
package ascii

const (
	nilByte = 0x00

	printableBegin = 0x20
	printableEnd   = 0x80

	extended = 0xff
)

// Is reports whether s contains only ASCII characters (0-127).
func Is(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i]&printableEnd != 0 {
			return false
		}
	}

	return true
}

// IsPrintable reports whether s contains only printable ASCII characters
// (32-127).
func IsPrintable(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < printableBegin || s[i]&printableEnd != 0 {
			return false
		}
	}

	return true
}

// IsExtended reports whether s contains only extended ASCII characters (0-255).
func IsExtended(s string) bool {
	for _, r := range s {
		if r > extended {
			return false
		}
	}

	return true
}

// HasNil reports whether s contains a null byte.
func HasNil(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == nilByte {
			return true
		}
	}
	return false
}
