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

package main

import (
	"bytes"
	"fmt"
	"log"

	"gitlab.com/iglou.eu/goulc/jsonl"
)

// LogEntry represents a structured log line that we want to serialize and deserialize.
type LogEntry struct {
	Level   string `json:"level"`
	Message string `json:"message"`
	Code    int    `json:"code,omitempty"`
}

func main() {
	// 1. Marshal records to JSONL
	entries := []LogEntry{
		{Level: "info", Message: "Application started", Code: 0},
		{Level: "warning", Message: "Memory usage high", Code: 42},
		{Level: "error", Message: "Failed to connect to database"},
	}

	encoded, err := jsonl.Marshal(entries)
	if err != nil {
		log.Fatalf("Failed to marshal JSONL: %v", err)
	}

	fmt.Printf("--- Marshaled JSONL ---\n%s\n", string(encoded))

	// 2. Unmarshal records from JSONL
	jsonData := []byte(`{"level":"info","message":"User logged in"}
{"level":"error","message":"Invalid token","code":401}
`)

	decoded, err := jsonl.Unmarshal[LogEntry](jsonData)
	if err != nil {
		log.Fatalf("Failed to unmarshal JSONL: %v", err)
	}

	fmt.Printf("--- Unmarshaled JSONL ---\n")
	for i, entry := range decoded {
		fmt.Printf("Record %d: Level=%s, Message=%s, Code=%d\n", i+1, entry.Level, entry.Message, entry.Code)
	}

	// 3. Unmarshal from an io.Reader (streaming, memory-efficient)
	jsonlStream := bytes.NewReader([]byte(`{"level":"debug","message":"Streaming parser initialized"}
{"level":"info","message":"Processing batch 42","code":200}
`))

	streamed, err := jsonl.UnmarshalStream[LogEntry](jsonlStream)
	if err != nil {
		log.Fatalf("Failed to unmarshal stream: %v", err)
	}

	fmt.Printf("\n--- Streamed JSONL ---\n")
	for i, entry := range streamed {
		fmt.Printf("Stream Record %d: Level=%s, Message=%s, Code=%d\n", i+1, entry.Level, entry.Message, entry.Code)
	}
}
