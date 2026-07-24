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

package bytesize

import "encoding"

// Compile-time interface conformance checks. They break the build if a
// method signature drifts from its contract.
var (
	_ encoding.TextMarshaler   = Size{}
	_ encoding.TextUnmarshaler = (*Size)(nil)
)

// MarshalText implements encoding.TextMarshaler using the canonical IEC
// string representation. It lets a Size act as a JSON map key and
// integrate with text-based tooling such as flag.TextVar.
func (s Size) MarshalText() ([]byte, error) {
	return []byte(s.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler. It accepts any value
// accepted by Parse.
func (s *Size) UnmarshalText(text []byte) error {
	size, err := New(string(text))
	if err != nil {
		return err
	}

	*s = size

	return nil
}
