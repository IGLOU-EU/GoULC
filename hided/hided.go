/*
 * Copyright 2024 Adrien Kara
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

// Package hided provides types and methods to obfuscate or mask sensitive data
// It can be used to ensure that sensitive information is not exposed in logs or any outputs
package hided

import "fmt"

// Hider defines types that can be obfuscated
type Hider interface {
	// String returns the obfuscated string (expected output: "***")
	fmt.Stringer

	// HashMD5 returns an MD5 hashed representation for obfuscation comparison
	// Note: MD5 is used only for obfuscation, not for cryptographic security
	HashMD5() string

	// Value returns the underlying value
	Value() any
}
