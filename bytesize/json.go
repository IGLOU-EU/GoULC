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

package bytesize

import (
	"encoding/json"
	"errors"
)

// Compile-time interface conformance checks. They break the build if a
// method signature drifts from its contract.
var (
	_ json.Marshaler   = Size{}
	_ json.Unmarshaler = (*Size)(nil)
)

// ErrJSONInvalidType reports a JSON value that is neither a string nor a
// number. Callers match it with errors.Is.
var ErrJSONInvalidType = errors.New(
	"invalid JSON byte size type, it should be a string or a number")

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *Size) UnmarshalJSON(b []byte) error {
	var i any
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}

	switch value := i.(type) {
	// Accordingly with https://pkg.go.dev/encoding/json#Unmarshal
	// JSON numbers are always considered as an interface value of float64.
	case float64:
		// Check the int64 range and keep the fractional part, so the
		// number branch behaves exactly like the string branch.
		if err := integerOverflow(value); err != nil {
			return err
		}

		*d = Size{t: int64(value), f: value, r: ToString(value)}
	case string:
		var err error
		*d, err = New(value)
		if err != nil {
			return err
		}
	default:
		return ErrJSONInvalidType
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (b Size) MarshalJSON() ([]byte, error) {
	return []byte(`"` + b.String() + `"`), nil
}
