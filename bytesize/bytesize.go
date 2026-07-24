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

// Package bytesize provides functionality for handling byte size values using
// IEC binary units (powers of 1024). It offers parsing, formatting, and
// arithmetic operations for byte sizes from Bytes up to Pebibytes.
//
// The package focuses on IEC binary units (KiB, MiB, etc.)
// rather than SI decimal units (KB, MB, etc.)
// to avoid ambiguity in size representations. It supports:
//   - Parsing size strings with units (e.g., "42.5MiB", "1.2GiB")
//   - Converting between different units
//   - Basic arithmetic operations on sizes
//   - Handling negative and floating-point values
//   - Range limited to what int64 can represent (approximately 8 EiB)
package bytesize

import (
	"errors"
	"math"
	"strconv"
)

// Size represents a byte size value with both truncated integer and exact
// floating-point representations and stores the IEC string representation of
// the size.
type Size struct {
	// t holds the size as a truncated int64 value in bytes.
	// Truncation is toward zero without rounding to avoid potential overflows
	// and maintain consistent behavior across positive and negative values.
	// For example, "42.99" => 42, "-42.99" => -42
	t int64

	// f holds the exact size as a float64 value in bytes.
	// This preserves the fractional component for accurate calculations
	// and string representations.
	f float64

	// r stores the canonical IEC string representation of the size.
	// This ensures consistent formatting and unit display (e.g., "42.5MiB").
	r string
}

const (
	// Base IEC binary units in bytes, each a power of 1024
	Byte int64 = 1 << (10 * iota) // 1 byte
	Kibi
	Mebi
	Gibi
	Tebi
	Pebi

	// multiplier is the base for IEC binary units (1024)
	multiplier = Kibi

	// IEC binary unit symbols
	ByteSymbol = "B"   // Byte
	KibiSymbol = "KiB" // Kibibyte
	MebiSymbol = "MiB" // Mebibyte
	GibiSymbol = "GiB" // Gibibyte
	TebiSymbol = "TiB" // Tebibyte
	PebiSymbol = "PiB" // Pebibyte

	percent = 100
	bitSize = 64
)

// Sentinel errors returned by Parse and the Size methods.
// Callers match them with errors.Is.
var (
	// ErrEmptyString reports an empty size string.
	ErrEmptyString = errors.New("size string cannot be empty")

	// ErrNoValue reports a size string with no leading numeric value.
	ErrNoValue = errors.New("no numeric value found in the size string")

	// ErrInvalidIEC reports an unrecognized IEC unit symbol.
	ErrInvalidIEC = errors.New("invalid IEC unit symbol in the size string")

	// ErrIntegerOverflow reports a value that cannot be represented as an
	// int64 count of bytes.
	ErrIntegerOverflow = errors.New(
		"size value cannot be represented as an int64")
)

// ByteValueIEC contains the byte values for each IEC binary unit in
// ascending order. This slice is used internally for unit conversion
// and formatting.
var ByteValueIEC = [...]int64{
	Byte,
	Kibi,
	Mebi,
	Gibi,
	Tebi,
	Pebi,
}

// ByteSymbolIEC contains the string symbols for each IEC binary unit
// in ascending order. The index of each symbol corresponds to the same
// index in ByteValueIEC.
var ByteSymbolIEC = [...]string{
	ByteSymbol,
	KibiSymbol,
	MebiSymbol,
	GibiSymbol,
	TebiSymbol,
	PebiSymbol,
}

// Parse parses a string representation of a byte size with IEC binary units
// and returns its components.
//
// It accepts strings in the format "NUMBER[UNIT]" where:
//   - number can be an integer, floating-point, or negative value
//   - unit is an optional IEC binary unit (B, KiB, MiB, GiB, TiB, PiB)
//
// Truncated value are simple truncation toward zero with no rounding.
//   - We don't want to risk an overflow by rounding to the nearest value
//     (-2.5 => -2, 2.5 => 3) or flooring to the nearest negative value
//     (-2.5 => -3, 2.5 => 2).
//   - We "can't" use partial byte like "1.5 Bytes = 8 bits + 4 bits",
//
// that require to set an arbitrary Byte width, which is most likely not what
// you want.
//
// The function returns:
//   - truncated: the size in bytes as an int64, truncated toward zero
//   - fractional: the exact size in bytes as a float64
//   - representation: the canonical IEC string representation
//   - error: if the input is invalid or the value is too large
//
// Examples:
//
//	Parse("42.42M") returns (44480593, 44480593.92, "42.42MiB", nil)
//	Parse("1024") returns (1024, 1024.0, "1KiB", nil)
//	Parse("-2.5KiB") returns (-2560, -2560.0, "-2.5KiB", nil)
//
// If a short unit symbol is used (e.g., "M" instead of "MiB"),
// it is automatically converted to the canonical IEC to avoid ambiguity
// with SI decimal units.
func Parse(s string) (
	truncated int64, fractional float64, representation string, err error,
) {
	if s == "" {
		return 0, 0, "", ErrEmptyString
	}

	if s == "0" {
		return 0, 0, "0B", nil
	}

	// Find the position of the first uppercase letter to split the size
	// and the symbol (if any). Every IEC symbol starts with an uppercase
	// ASCII letter, including the bare "B" byte unit.
	runePos := -1
	for i := range s {
		if s[i] < 'A' || s[i] > 'Z' {
			continue
		}

		runePos = i
		break
	}

	// Set raw values for size and symbol
	sizeRaw := s
	symbolRaw := ""

	if runePos == 0 {
		return 0, 0, "", ErrNoValue
	}

	if runePos > 0 {
		sizeRaw = s[0:runePos]
		symbolRaw = s[runePos:]
	}

	// Convert the string size to a float
	size, err := strconv.ParseFloat(sizeRaw, bitSize)
	if err != nil {
		return 0, 0, "", err
	}

	// Without a symbol we assume it's in bytes. The range check also
	// rejects non-finite values (NaN, +/-Inf) accepted by ParseFloat.
	if symbolRaw == "" {
		if err := integerOverflow(size); err != nil {
			return 0, 0, "", err
		}

		return int64(size), size, ToString(size), nil
	}

	// Find the exponent of the symbol
	exponent, err := exponentFromSymbol(symbolRaw)
	if err != nil {
		return 0, 0, "", err
	}

	// Calculate the Byte size and check if it's too large
	result := size * float64(ByteValueIEC[exponent])
	if err := integerOverflow(result); err != nil {
		return 0, 0, "", err
	}

	// Return the result
	return int64(result), result, ToString(result), nil
}

// ToString returns a string representing the Size value in IEC format.
// It uses the most appropriate unit to keep the number human-readable.
// Non-finite values cannot be expressed in IEC units and are formatted
// as "+Inf", "-Inf" or "NaN".
func ToString(b float64) string {
	// Rejecting non-finite values up front keeps the unit selection loop
	// bounded, an infinite input would otherwise never divide down.
	if math.IsInf(b, 0) || math.IsNaN(b) {
		return strconv.FormatFloat(b, 'f', -1, bitSize)
	}

	if b == 0 {
		return "0" + ByteSymbol
	}

	var negative string
	if b < 0 {
		// We use the mathematical rule of double negation to get
		// the positive value
		b = -b

		// Store the negative sign for later
		negative = "-"
	}

	exponent := exponentFromSize(b)

	var value float64
	if exponent == 0 {
		value = math.Round(b)
	} else {
		value = math.Round(
			(b/float64(ByteValueIEC[exponent]))*percent) / percent
	}

	return negative +
		strconv.FormatFloat(
			value,
			'f', -1, bitSize,
		) +
		ByteSymbolIEC[exponent]
}

// New creates a new Size from a string representation.
// It uses Parse and returns a Size struct or an error if the input is invalid.
func New(s string) (Size, error) {
	t, f, r, err := Parse(s)
	if err != nil {
		return Size{}, err
	}

	return Size{t: t, f: f, r: r}, nil
}

// NewInt creates a new Size from an int64 value representing bytes.
// This is useful when you have a byte count and want to convert it to
// a human-readable format with appropriate IEC binary units.
func NewInt(i int64) Size {
	return Size{
		t: i,
		f: float64(i),
		r: ToString(float64(i)),
	}
}

// Bytes returns the Byte count of the Size as an int64.
// It is truncated toward zero across positive and negative values
// (e.g., "42.42MiB" => 44480593)
func (s Size) Bytes() int64 {
	return s.t
}

// Exact returns the floating-point value of the byte size as a float64.
// (e.g., "42.42MiB" => 44480593.92)
func (s Size) Exact() float64 {
	return s.f
}

// Add adds the given size string to the current Size.
// The size string must be in a valid format as accepted by Parse.
// Returns an error if the input string is invalid or if the result
// would overflow.
func (s *Size) Add(value string) error {
	size, err := New(value)
	if err != nil {
		return err
	}

	s.t += size.t
	s.f += size.f
	s.r = ToString(s.f)

	return integerOverflow(s.f)
}

// AddInt adds the given number of bytes to the current Size.
// This is a more efficient alternative to Add when working with
// raw byte counts.
func (s *Size) AddInt(i int64) error {
	s.t += i
	s.f += float64(i)
	s.r = ToString(s.f)

	return integerOverflow(s.f)
}

// String is the stringer method for the Size struct.
// It returns the canonical IEC string representation of the Size.
func (s Size) String() string {
	if s.r == "" {
		return ToString(s.f)
	}

	return s.r
}

// exponentFromSize determines the appropriate IEC binary for a given byte size.
// It returns the index into ByteValueIEC/ByteSymbolIEC arrays corresponding to
// the largest unit that can represent the size.
func exponentFromSize(size float64) int {
	if size == 0 {
		return 0
	}

	// Find the exponent of the largest unit by dividing the size by
	// the multiplier until the size is less than the multiplier.
	var exp int
	for i := size; i >= float64(multiplier); i /= float64(multiplier) {
		exp++
	}

	// Ensure the exponent doesn't exceed our largest available unit.
	maxExp := len(ByteValueIEC) - 1
	if exp > maxExp {
		return maxExp
	}

	return exp
}

// exponentFromSymbol converts an IEC binary unit symbol to its corresponding
// exponent (index in ByteValueIEC/ByteSymbolIEC arrays).
//
// It handles both full IEC symbols (e.g., "MiB") and short forms (e.g., "M"),
// converting them to the canonical IEC form. Returns an error if the symbol
// is not recognized.
func exponentFromSymbol(symbol string) (int, error) {
	short := len(symbol) == 1

	for i, v := range ByteSymbolIEC {
		if !short && v == symbol {
			return i, nil
		}

		if short && symbol[0] == v[0] {
			return i, nil
		}
	}

	return 0, ErrInvalidIEC
}

// integerOverflow checks whether a byte size value can be represented as an
// int64, whose range tops out just below 8 EiB (2^63 bytes). Values outside
// that range and non-finite values (NaN, +/-Inf) are rejected, so a checked
// value is always safe to convert with int64().
//
// Returns ErrIntegerOverflow if the size is not representable, nil otherwise.
func integerOverflow(size float64) error {
	// float64(math.MaxInt64) rounds up to exactly 2^63, one past the last
	// valid int64, so the upper bound must be exclusive. The lower bound
	// float64(math.MinInt64) is exactly -2^63, a valid int64, and stays
	// inclusive. NaN is rejected explicitly because it escapes every
	// ordered comparison.
	if math.IsNaN(size) ||
		size >= float64(math.MaxInt64) || size < float64(math.MinInt64) {
		return ErrIntegerOverflow
	}

	return nil
}
