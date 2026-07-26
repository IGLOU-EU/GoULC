/* model_test.go
 *
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

package model

import (
	"encoding/json"
	"testing"
)

func TestConfig_JSONTags(t *testing.T) {
	cfg := Config{
		Colored:     true,
		AddSource:   true,
		ForceSyslog: true,
		Level:       "DEBUG",
		TimeFormat:  "2006-01-02",
		Cancel:      func() {},
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	expectedKeys := []string{"colored", "add_source", "force_syslog", "level", "time_format"}
	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("expected JSON key %q not found", key)
		}
	}

	if _, ok := raw["Cancel"]; ok {
		t.Error("Cancel field should not appear in JSON output (tagged json:\"-\")")
	}

	if len(raw) != len(expectedKeys) {
		t.Errorf("expected %d JSON keys, got %d", len(expectedKeys), len(raw))
	}
}

func TestConfig_OmitZero(t *testing.T) {
	var cfg Config

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	if got, want := string(data), "{}"; got != want {
		t.Errorf("zero Config marshals to %s, want %s", got, want)
	}
}

func TestConfig_ZeroValue(t *testing.T) {
	var cfg Config

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"Colored", cfg.Colored, false},
		{"AddSource", cfg.AddSource, false},
		{"ForceSyslog", cfg.ForceSyslog, false},
		{"Level", cfg.Level, ""},
		{"TimeFormat", cfg.TimeFormat, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("zero-value Config.%s = %v, want %v", tt.name, tt.got, tt.want)
			}
		})
	}

	if cfg.Cancel != nil {
		t.Error("zero-value Config.Cancel should be nil")
	}
}

func TestWriter_ZeroValue(t *testing.T) {
	var w Writer

	if w.Out != nil {
		t.Error("zero-value Writer.Out should be nil")
	}
	if w.Err != nil {
		t.Error("zero-value Writer.Err should be nil")
	}
}
