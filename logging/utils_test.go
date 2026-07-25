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

package logging

import (
	"log/slog"
	"runtime"
	"strings"
	"testing"
)

func TestSetColorLevel(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
		want  color
	}{
		{"Debug", slog.LevelDebug, colorBrightGrey},
		{"Info", slog.LevelInfo, colorCyan},
		{"Warn", slog.LevelWarn, colorBrightYellow},
		{"Error", slog.LevelError, colorRed},
		{"Unknown", slog.Level(42), colorMagenta},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := setColorLevel(tt.level)
			if got != tt.want {
				t.Errorf("setColorLevel(%v) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

func TestSyslogPrefixCache(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
	}{
		{"Debug", slog.LevelDebug},
		{"Info", slog.LevelInfo},
		{"Warn", slog.LevelWarn},
		{"Error", slog.LevelError},
		{"Unknown", slog.Level(42)},
		{"AboveError", slog.LevelError + 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := syslogPrefix(tt.level)

			want := BuildSyslogPrefix(tt.level)
			if got != want {
				t.Errorf("syslogPrefix(%v) = %q, want %q", tt.level, got, want)
			}
		})
	}
}

func TestSource(t *testing.T) {
	// Capture a real PC from this call site.
	var pcs [1]uintptr
	runtime.Callers(1, pcs[:])
	pc := pcs[0]

	// Determine the full file path of this test file via runtime so we
	// can derive a basePath to trim.
	fs := runtime.CallersFrames([]uintptr{pc})
	frame, _ := fs.Next()
	fullFile := frame.File // e.g. "/home/.../logging/utils_test.go"

	// Use everything up to and including "logging/" as basePath.
	idx := strings.LastIndex(fullFile, "logging/")
	if idx == -1 {
		t.Fatalf("unexpected file path: %s", fullFile)
	}
	basePath := fullFile[:idx]

	src := source(basePath, pc)

	if !strings.HasPrefix(src.File, "logging/") {
		t.Errorf("expected File to start with \"logging/\", got %q", src.File)
	}
	if src.Line <= 0 {
		t.Errorf("expected Line > 0, got %d", src.Line)
	}
}
