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

package hided

import (
	"crypto/md5"
	"encoding/hex"
)

// String holds sensitive data and implements obfuscation
type String string

// String implements fmt.Stringer to return an obfuscated string
func (_ String) String() string {
	return "***"
}

// HashMD5 returns an MD5 hash of the string for obfuscation comparison
// Note: MD5 is used solely for obfuscation, not for security
func (s String) HashMD5() string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}

// Value returns the underlying string value
func (s String) Value() any {
	return string(s)
}
