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

package bytesize

import (
	"encoding/json"
	"errors"
)

const (
	ErrJsonInvalidType = "invalid JSON byte Size type, it should be a string or a number"
)

// UnmarshalJSON implements the json.Unmarshaler interface.
func (d *Size) UnmarshalJSON(b []byte) error {
	var i interface{}
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}

	switch value := i.(type) {
	// Accordingly with https://pkg.go.dev/encoding/json#Unmarshal
	// JSON numbers are always considered as ab interface value of float64.
	case float64:
		*d = NewInt(int64(value))
	case string:
		var err error
		*d, err = New(value)
		if err != nil {
			return err
		}
	default:
		return errors.New(ErrJsonInvalidType)
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
func (b Size) MarshalJSON() ([]byte, error) {
	return []byte(`"` + b.String() + `"`), nil
}
