/*
 * Copyright 2025 Adrien Kara
 *
 * This file is part of GoULC.
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
 * SPDX-License-Identifier: LGPL-3.0-or-later
 */

// Package ascii implements ASCII string validation functions.
package ascii

const (
	nilByte = 0x00

	// Printable ASCII stops at tilde (0x7e), DEL (0x7f) is a control
	// character.
	printableBegin = 0x20
	printableMax   = 0x7e

	asciiMax = 0x7f
)

// Is reports whether s contains only ASCII characters (0-127).
func Is(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > asciiMax {
			return false
		}
	}

	return true
}

// IsPrintable reports whether s contains only printable ASCII characters
// (32-126).
func IsPrintable(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < printableBegin || s[i] > printableMax {
			return false
		}
	}

	return true
}

// IsExtended reports whether s contains at least one extended ASCII byte
// (128-255). Unlike Is and IsPrintable, which validate that every byte stays
// within a range, IsExtended detects the presence of a single byte outside
// the standard 7-bit ASCII set.
func IsExtended(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > asciiMax {
			return true
		}
	}

	return false
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
