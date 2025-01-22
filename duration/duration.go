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

package duration

import (
	"encoding/json"
	"errors"
	"time"
)

const (
	ErrDurationInvalidType = "invalid JSON duration type, it should be a number or a string"
)

// Duration is a custom type that wraps time.Duration to provide
// customized JSON serialization and deserialization.
type Duration struct {
	time.Duration
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// It supports integer, float, and string representations of durations.
// With string inputs, it uses time.ParseDuration to interpret standard
// duration formats (e.g., "1h30m").
func (d *Duration) UnmarshalJSON(b []byte) error {
	var i interface{}
	if err := json.Unmarshal(b, &i); err != nil {
		return err
	}

	switch value := i.(type) {
	// Accordingly with https://pkg.go.dev/encoding/json#Unmarshal
	// JSON numbers are always considered as ab interface value of float64.
	case float64:
		d.Duration = time.Duration(value)
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return errors.Join(errors.New("failed to parse duration "+value), err)
		}
	default:
		return errors.New(ErrDurationInvalidType)
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
// The duration is serialized as a string in the format accepted by
// time.ParseDuration.
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.String() + `"`), nil
}

// ToTimeDuration returns the underlying time.Duration value.
// It is useful in cases where you need to work with a copy of
// the time.Duration type.
func (d Duration) ToTimeDuration() time.Duration {
	return d.Duration
}
