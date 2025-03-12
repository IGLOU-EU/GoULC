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
//   - Range limited to what int64 can represent (approximately 9 EiB)
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

	// Error messages
	ErrEmptyString     = "A Size string value cannot be empty"
	ErrNoValue         = "No numeric value found in the given string"
	ErrInvalidIEC      = "Invalid IEC unit symbol in Size string"
	ErrIntegerOverflow = "Size value is too large to be represented as an int64"

	percent  = 100
	bitSize  = 64
	exponent = 10
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
		return 0, 0, "", errors.New(ErrEmptyString)
	}

	if s == "0" {
		return 0, 0, "0B", nil
	}

	// Find the position of the first uppercase letter
	// To split the size and the symbol (if any)
	runePos := -1
	for i := range s {
		if s[i] < 'G' || s[i] > 'Z' {
			continue
		}

		runePos = i
		break
	}

	// Set raw values for size and symbol
	sizeRaw := s
	symbolRaw := ""

	if runePos == 0 {
		return 0, 0, "", errors.New(ErrNoValue)
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

	// Without a symbol we assume it's in bytes
	if symbolRaw == "" {
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
func ToString(b float64) string {
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

	return Size{t, f, r}, nil
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
	max := len(ByteValueIEC) - 1
	if exp > max {
		return max
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

	return 0, errors.New(ErrInvalidIEC)
}

// integerOverflow checks if a byte size value exceeds the maximum
// representable value. This is used to prevent integer overflow when working
// with large sizes. The maximum value is slightly less than 9 EiB
// (9 * 2^60 bytes), which is the largest value that can be safely
// represented by an int64.
//
// Returns ErrIntegerOverflow if the size is too large, nil otherwise.
func integerOverflow(size float64) error {
	if math.Ldexp(size, exponent) >= 1<<bitSize {
		return errors.New(ErrIntegerOverflow)
	}

	return nil
}
