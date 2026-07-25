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

// Package jsonl encodes and decodes JSON Lines (JSONL) data.
package jsonl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

const (
	lineSeparator = '\n'
	heuristicSize = 512

	// maxRecordsHint caps the slice preallocation of Unmarshal: the
	// line count is known before any line is validated, so the hint
	// must not scale with attacker-controlled input.
	maxRecordsHint = 1024
)

// DefaultMaxLineSize is the line size bound applied by UnmarshalStream.
// It is generous on purpose: it only exists to keep a hostile or
// corrupted stream from buffering unbounded data in memory.
const DefaultMaxLineSize = 16 << 20 // 16 MiB

// Errors reported on invalid JSONL input. They are returned wrapped
// with the offending line number, match them with errors.Is.
var (
	// ErrBlankLine reports a blank line inside JSONL data: the JSON
	// Lines spec requires every line to hold a JSON value.
	// https://jsonlines.org/#each-line-is-a-valid-json-value
	ErrBlankLine = errors.New("blank line inside JSONL data")

	// ErrLineTooLong reports a line exceeding the maximum line size
	// accepted by UnmarshalStream or UnmarshalStreamLimit.
	ErrLineTooLong = errors.New("line exceeds maximum size")
)

// Marshal encodes values as JSON Lines (JSONL). Each value is marshaled
// as a single JSON object, separated by newlines. A nil or empty slice
// returns a nil byte slice.
func Marshal[T any](records []T) ([]byte, error) {
	if len(records) == 0 {
		return nil, nil
	}

	var buf bytes.Buffer
	// Rough heuristic bytes per record
	buf.Grow(len(records) * heuristicSize)

	enc := json.NewEncoder(&buf)
	for i := range records {
		if err := enc.Encode(records[i]); err != nil {
			return nil, fmt.Errorf("jsonl: line %d: %w", i+1, err)
		}
	}

	return buf.Bytes(), nil
}

// Unmarshal parses JSON Lines (JSONL) data into a slice of typed records.
func Unmarshal[T any](data []byte) ([]T, error) {
	if len(data) == 0 {
		return nil, nil
	}

	if l := len(data); data[l-1] == lineSeparator {
		data = data[:l-1]
	}

	lines := bytes.Count(data, []byte{lineSeparator}) + 1
	records := make([]T, 0, min(lines, maxRecordsHint))

	n := 0
	for line := range bytes.SplitSeq(data, []byte{lineSeparator}) {
		n++
		if len(line) == 0 {
			return nil, fmt.Errorf("jsonl: line %d: %w", n, ErrBlankLine)
		}

		var rec T
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, fmt.Errorf("jsonl: line %d: %w", n, err)
		}
		records = append(records, rec)
	}

	return records, nil
}

// UnmarshalStream parses JSON Lines (JSONL) data from an io.Reader
// into a slice of typed records. Lines longer than DefaultMaxLineSize
// are rejected with ErrLineTooLong, use UnmarshalStreamLimit to pick
// another bound.
func UnmarshalStream[T any](r io.Reader) ([]T, error) {
	return UnmarshalStreamLimit[T](r, DefaultMaxLineSize)
}

// UnmarshalStreamLimit parses JSON Lines (JSONL) data from an io.Reader
// into a slice of typed records. maxLineSize bounds, in bytes, the size
// of a single line: longer lines abort parsing with ErrLineTooLong so a
// hostile or corrupted stream cannot buffer unbounded data in memory.
// A maxLineSize <= 0 falls back to DefaultMaxLineSize.
func UnmarshalStreamLimit[T any](r io.Reader, maxLineSize int) ([]T, error) {
	if maxLineSize <= 0 {
		maxLineSize = DefaultMaxLineSize
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(nil, maxLineSize)

	var records []T

	n := 0
	for scanner.Scan() {
		n++
		line := scanner.Bytes()
		if len(line) == 0 {
			return nil, fmt.Errorf("jsonl: line %d: %w", n, ErrBlankLine)
		}

		var rec T
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, fmt.Errorf("jsonl: line %d: %w", n, err)
		}
		records = append(records, rec)
	}

	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return nil, fmt.Errorf("jsonl: line %d: %w", n+1, ErrLineTooLong)
		}
		return nil, err
	}

	return records, nil
}
