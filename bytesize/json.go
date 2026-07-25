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
	"bytes"
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
// A JSON number and a JSON string go through the exact same Parse path, so
// 1.9 and "1.9" yield the same Size and out-of-range values are rejected
// identically. Decoding numbers as json.Number (instead of the default
// float64) keeps the literal untouched, a float64 would drop the fraction
// and silently wrap on overflow.
func (d *Size) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()

	var i any
	if err := dec.Decode(&i); err != nil {
		return err
	}

	var raw string
	switch value := i.(type) {
	case json.Number:
		raw = value.String()
	case string:
		raw = value
	default:
		return ErrJSONInvalidType
	}

	size, err := New(raw)
	if err != nil {
		return err
	}

	*d = size

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (b Size) MarshalJSON() ([]byte, error) {
	return []byte(`"` + b.String() + `"`), nil
}
