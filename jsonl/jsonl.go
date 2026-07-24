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

const lineSeparator = '\n'
const heuristicSize = 512

// ErrBlankLine reports a blank line inside JSONL data: the JSON Lines
// spec requires every line to hold a JSON value.
// https://jsonlines.org/#each-line-is-a-valid-json-value
// It is returned wrapped with the offending line number, match it with
// errors.Is.
var ErrBlankLine = errors.New("blank line inside JSONL data")

// Marshal encodes values as JSON Lines (JSONL). Each value is marshaled
// as a single JSON object, separated by newlines. A nil or empty slice
// returns an empty byte slice.
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

// Unmarshal parses JSON Lines (JSONL) data into a slice of typed records
func Unmarshal[T any](data []byte) ([]T, error) {
	if len(data) == 0 {
		return nil, nil
	}

	if l := len(data); data[l-1] == lineSeparator {
		data = data[:l-1]
	}

	lines := bytes.Count(data, []byte{lineSeparator}) + 1
	records := make([]T, 0, lines)

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

// UnmarshalStream parses JSON Lines (JSONL) data
// from an io.Reader into a slice of typed records
func UnmarshalStream[T any](r io.Reader) ([]T, error) {
	reader := bufio.NewReader(r)

	var records []T

	n := 0
	for {
		line, err := reader.ReadBytes(lineSeparator)
		if err == io.EOF && len(line) == 0 {
			break
		}

		// Trim the line separator from the end
		if len(line) > 0 && line[len(line)-1] == lineSeparator {
			line = line[:len(line)-1]
		}

		// Handle trailing newline on last line: if we hit EOF and the line
		// is empty after trimming, we are done.
		if len(line) == 0 && err == io.EOF {
			break
		}

		n++
		if len(line) == 0 {
			return nil, fmt.Errorf("jsonl: line %d: %w", n, ErrBlankLine)
		}

		var rec T
		if err := json.Unmarshal(line, &rec); err != nil {
			return nil, fmt.Errorf("jsonl: line %d: %w", n, err)
		}
		records = append(records, rec)

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return records, nil
}
