/*
 * Copyright 2025 Adrien Kara
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

package hided

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// String holds sensitive data and implements obfuscation.
// It is a struct with an unexported field to prevent direct access or casting.
type String struct {
	val []byte
}

var _ = Hider(String{})
var _ = json.Marshaler(String{})
var _ = json.Unmarshaler(&String{})

// NewString creates a new String from the given plaintext.
func NewString(s string) String {
	return String{val: []byte(s)}
}

// String implements fmt.Stringer to return an obfuscated string.
func (_ String) String() string {
	return obfuscated
}

// GoString implements fmt.GoStringer to prevent leaks via %#v.
func (_ String) GoString() string {
	return obfuscated
}

// Format implements fmt.Formatter to prevent leaks via any fmt verb.
func (_ String) Format(f fmt.State, _ rune) {
	fmt.Fprint(f, obfuscated)
}

// IsEmpty returns true if the underlying string is empty.
func (s String) IsEmpty() bool {
	return len(s.val) == 0
}

// IsZero reports whether the String holds no secret. It unlocks the json
// ",omitzero" tag option: omitzero is evaluated before MarshalJSON, so an
// empty secret is omitted from the output instead of appearing as "***".
func (s String) IsZero() bool {
	return s.IsEmpty()
}

// HashMD5 returns an MD5 hash of the string for obfuscation comparison.
// Note: MD5 is used solely for obfuscation, not for security.
func (s String) HashMD5() string {
	hash := md5.Sum(s.val)
	return hex.EncodeToString(hash[:])
}

// Value returns the underlying string value.
func (s String) Value() any {
	return string(s.val)
}

// Reveal returns the underlying plaintext as a typed string. It is the
// direct accessor for call sites that statically hold a String, where the
// generic Value[T] indirection through the Hider interface brings nothing.
func (s String) Reveal() string {
	return string(s.val)
}

// MarshalJSON implements json.Marshaler to prevent leaks in JSON output.
func (_ String) MarshalJSON() ([]byte, error) {
	return json.Marshal(obfuscated)
}

// MarshalText implements encoding.TextMarshaler to prevent leaks in text output.
func (_ String) MarshalText() ([]byte, error) {
	return []byte(obfuscated), nil
}

// UnmarshalJSON implements json.Unmarshaler to populate the hidden value from
// a JSON string.
func (s *String) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	s.val = []byte(raw)
	return nil
}

// UnmarshalText implements encoding.TextUnmarshaler to populate the hidden
// value from text.
func (s *String) UnmarshalText(data []byte) error {
	s.val = make([]byte, len(data))
	copy(s.val, data)
	return nil
}
