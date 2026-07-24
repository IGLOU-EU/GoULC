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

// Package duration wraps time.Duration with JSON serialization support.
//
// In JSON, a duration is either a string in time.ParseDuration format
// (e.g. "1h30m") or a bare number of nanoseconds, the unit of
// time.Duration itself.
package duration

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gitlab.com/iglou.eu/goulc/contract"
)

var (
	// ErrBadDuration wraps any parse failure of a duration value, so
	// callers can match it with errors.Is.
	ErrBadDuration = errors.New("invalid duration")
	// ErrDurationInvalidType reports a JSON value whose type cannot
	// represent a duration.
	ErrDurationInvalidType = errors.New(
		"invalid JSON duration type, it should be a number or a string")
)

// Duration is a custom type that wraps time.Duration to provide
// customized JSON serialization and deserialization.
type Duration struct {
	time.Duration
}

// Compile-time conformance proofs, so any signature drift breaks the
// build. fmt.Stringer is promoted from the embedded time.Duration.
var (
	_ json.Marshaler           = Duration{}
	_ json.Unmarshaler         = (*Duration)(nil)
	_ encoding.TextMarshaler   = Duration{}
	_ encoding.TextUnmarshaler = (*Duration)(nil)
	_ fmt.Stringer             = Duration{}
	_ contract.IsZeroer        = Duration{}
)

// New wraps a time.Duration in a Duration.
func New(d time.Duration) Duration {
	return Duration{Duration: d}
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// A JSON string is parsed with time.ParseDuration (e.g. "1h30m"). A bare
// JSON number is a count of nanoseconds, the unit of time.Duration itself,
// and must be a whole number fitting in an int64: anything else (fraction,
// scientific notation, out-of-range value) returns an error wrapping
// ErrBadDuration. A JSON null leaves the value unchanged, per the
// encoding/json convention.
func (d *Duration) UnmarshalJSON(b []byte) error {
	// Per encoding/json convention, unmarshaling null is a no-op.
	if string(b) == "null" {
		return nil
	}

	// Decode numbers as json.Number instead of float64: float64 loses
	// precision above 2^53 and silently wraps values overflowing int64.
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()

	var i any
	if err := dec.Decode(&i); err != nil {
		return err
	}

	switch value := i.(type) {
	case json.Number:
		ns, err := value.Int64()
		if err != nil {
			return fmt.Errorf("%w: %q: %w", ErrBadDuration, value, err)
		}
		d.Duration = time.Duration(ns)
	case string:
		return d.UnmarshalText([]byte(value))
	default:
		return ErrDurationInvalidType
	}

	return nil
}

// MarshalJSON implements the json.Marshaler interface.
// The duration is serialized as a string in the format accepted by
// time.ParseDuration.
func (d Duration) MarshalJSON() ([]byte, error) {
	return []byte(`"` + d.String() + `"`), nil
}

// MarshalText implements the encoding.TextMarshaler interface, so a
// Duration can be used where a text form is expected, such as a JSON map
// key. The output is the time.Duration.String format.
func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface. The
// text is parsed with time.ParseDuration.
func (d *Duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return fmt.Errorf("%w: %q: %w", ErrBadDuration, text, err)
	}

	d.Duration = parsed

	return nil
}

// IsZero reports whether the duration is zero. encoding/json relies on it
// to honor the ",omitzero" struct tag option (Go 1.24+).
func (d Duration) IsZero() bool {
	return d.Duration == 0
}

// ToTimeDuration returns the underlying time.Duration value.
// It is useful in cases where you need to work with a copy of
// the time.Duration type.
func (d Duration) ToTimeDuration() time.Duration {
	return d.Duration
}
