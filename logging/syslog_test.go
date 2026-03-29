//go:build !windows

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
	"log/syslog"
	"os"
	"testing"
)

func TestPriority_SyslogPrefix(t *testing.T) {
	tests := []struct {
		name     string
		priority Priority
		want     string
	}{
		{"zero", Priority(0), "<0>"},
		{"info", Priority(6), "<6>"},
		{"debug", Priority(7), "<7>"},
		{"warning", Priority(4), "<4>"},
		{"error", Priority(3), "<3>"},
		{"large value", Priority(191), "<191>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.priority.SyslogPrefix()
			if got != tt.want {
				t.Errorf("Priority(%d).SyslogPrefix() = %q, want %q", tt.priority, got, tt.want)
			}
		})
	}
}

func TestIsSyslog(t *testing.T) {
	tests := []struct {
		name   string
		setEnv bool
		envVal string
		want   bool
	}{
		{"set", true, "8:12345", true},
		{"unset", false, "", false},
		{"empty", true, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv("JOURNAL_STREAM", tt.envVal)
			} else {
				// t.Setenv records the original value for cleanup,
				// then we unset the variable to simulate absence.
				t.Setenv("JOURNAL_STREAM", "")
				os.Unsetenv("JOURNAL_STREAM")
			}

			got := IsSyslog()
			if got != tt.want {
				t.Errorf("IsSyslog() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSyslogSeverity(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
		want  Priority
	}{
		{"debug", slog.LevelDebug, Priority(syslog.LOG_DEBUG)},
		{"info", slog.LevelInfo, Priority(syslog.LOG_INFO)},
		{"warn", slog.LevelWarn, Priority(syslog.LOG_WARNING)},
		{"error", slog.LevelError, Priority(syslog.LOG_ERR)},
		{"unknown positive", slog.Level(42), Priority(syslog.LOG_INFO)},
		{"unknown negative", slog.Level(-10), Priority(syslog.LOG_INFO)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SyslogSeverity(tt.level)
			if got != tt.want {
				t.Errorf("SyslogSeverity(%v) = %d, want %d", tt.level, got, tt.want)
			}
		})
	}
}

func TestBuildSyslogPrefix(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
		want  string
	}{
		{"debug", slog.LevelDebug, "<7>"},
		{"info", slog.LevelInfo, "<6>"},
		{"warn", slog.LevelWarn, "<4>"},
		{"error", slog.LevelError, "<3>"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildSyslogPrefix(tt.level)
			if got != tt.want {
				t.Errorf("BuildSyslogPrefix(%v) = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}
