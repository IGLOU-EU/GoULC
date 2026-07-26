//go:build gorm

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
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm"

	"gitlab.com/iglou.eu/goulc/logging/model"
)

// mockGormHider implements hided.GormHider for testing ParamsFilter.
type mockGormHider struct {
	value  string
	hidden string
}

func (m mockGormHider) String() string { return m.value }
func (m mockGormHider) Hiding() string { return m.hidden }

// newTestGormLogger creates a GormLogger backed by a buffer for assertions.
func newTestGormLogger() (*GormLogger, *bytes.Buffer) {
	var buf bytes.Buffer
	w := &model.Writer{Out: &buf, Err: &buf}
	h := NewHandler(nil, w, &HandlerOptions{Config: model.Config{Colored: false}, HandlerOptions: slog.HandlerOptions{Level: slog.LevelDebug}})
	log := slog.New(h)
	gl := NewGormLogger(log)
	return gl, &buf
}

func TestNewGormLogger(t *testing.T) {
	gl, _ := newTestGormLogger()
	if gl == nil {
		t.Fatal("NewGormLogger returned nil")
	}
	if gl.log == nil {
		t.Fatal("NewGormLogger internal logger is nil")
	}
}

func TestGormLogger_LogMode(t *testing.T) {
	gl, _ := newTestGormLogger()
	result := gl.LogMode(0)
	if result != gl {
		t.Fatalf("LogMode expected same instance, got %v", result)
	}
}

func TestGormLogger_Info(t *testing.T) {
	gl, buf := newTestGormLogger()
	gl.Info(context.Background(), "connection %s on port %d", "localhost", 5432)

	out := buf.String()
	if !strings.Contains(out, "Database") {
		t.Errorf("expected output to contain %q, got: %s", "Database", out)
	}
	if !strings.Contains(out, "connection localhost on port 5432") {
		t.Errorf("expected output to contain formatted message, got: %s", out)
	}
	if !strings.Contains(out, "INFO") {
		t.Errorf("expected output to contain INFO level, got: %s", out)
	}
}

func TestGormLogger_Warn(t *testing.T) {
	gl, buf := newTestGormLogger()
	gl.Warn(context.Background(), "slow query %dms", 1500)

	out := buf.String()
	if !strings.Contains(out, "Database") {
		t.Errorf("expected output to contain %q, got: %s", "Database", out)
	}
	if !strings.Contains(out, "slow query 1500ms") {
		t.Errorf("expected output to contain formatted message, got: %s", out)
	}
	if !strings.Contains(out, "WARN") {
		t.Errorf("expected output to contain WARN level, got: %s", out)
	}
}

func TestGormLogger_Error(t *testing.T) {
	gl, buf := newTestGormLogger()
	gl.Error(context.Background(), "failed to %s: %v", "connect", errors.New("timeout"))

	out := buf.String()
	if !strings.Contains(out, "Database") {
		t.Errorf("expected output to contain %q, got: %s", "Database", out)
	}
	if !strings.Contains(out, "failed to connect: timeout") {
		t.Errorf("expected output to contain formatted message, got: %s", out)
	}
	if !strings.Contains(out, "ERROR") {
		t.Errorf("expected output to contain ERROR level, got: %s", out)
	}
}

func TestGormLogger_Trace(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectLevel   string
		expectMessage string
	}{
		{
			name:          "no_error",
			err:           nil,
			expectLevel:   "DEBUG",
			expectMessage: "Database trace",
		},
		{
			name:          "record_not_found",
			err:           gorm.ErrRecordNotFound,
			expectLevel:   "DEBUG",
			expectMessage: gorm.ErrRecordNotFound.Error(),
		},
		{
			name:          "other_error",
			err:           errors.New("deadlock detected"),
			expectLevel:   "ERROR",
			expectMessage: "deadlock detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gl, buf := newTestGormLogger()
			begin := time.Now().Add(-42 * time.Millisecond)
			fc := func() (string, int64) {
				return "SELECT * FROM users WHERE id = 1", 3
			}

			gl.Trace(context.Background(), begin, fc, tt.err)

			out := buf.String()
			if !strings.Contains(out, tt.expectLevel) {
				t.Errorf("expected level %q in output, got: %s", tt.expectLevel, out)
			}
			if !strings.Contains(out, tt.expectMessage) {
				t.Errorf("expected message %q in output, got: %s", tt.expectMessage, out)
			}
			if !strings.Contains(out, "elapsed") {
				t.Errorf("expected %q in output, got: %s", "elapsed", out)
			}
			if !strings.Contains(out, "SELECT * FROM users WHERE id = 1") {
				t.Errorf("expected SQL trace in output, got: %s", out)
			}
			if !strings.Contains(out, "rows affected") {
				t.Errorf("expected %q in output, got: %s", "rows affected", out)
			}
		})
	}
}

func TestGormLogger_ParamsFilter(t *testing.T) {
	tests := []struct {
		name       string
		sql        string
		params     []any
		wantSQL    string
		wantParams []any
	}{
		{
			name:       "no_sensitive_params",
			sql:        "SELECT * FROM users WHERE id = ?",
			params:     []any{42, "plain"},
			wantSQL:    "SELECT * FROM users WHERE id = ?",
			wantParams: []any{42, "plain"},
		},
		{
			name: "sensitive_param_hidden",
			sql:  "INSERT INTO users (name, secret) VALUES (?, ?)",
			params: []any{
				"alice",
				mockGormHider{value: "supersecret", hidden: "***"},
			},
			wantSQL:    "INSERT INTO users (name, secret) VALUES (?, ?)",
			wantParams: []any{"alice", "***"},
		},
		{
			name: "all_sensitive_params",
			sql:  "UPDATE secrets SET a = ?, b = ?",
			params: []any{
				mockGormHider{value: "val1", hidden: "***"},
				mockGormHider{value: "val2", hidden: "***"},
			},
			wantSQL:    "UPDATE secrets SET a = ?, b = ?",
			wantParams: []any{"***", "***"},
		},
		{
			name:       "empty_params",
			sql:        "SELECT 1",
			params:     []any{},
			wantSQL:    "SELECT 1",
			wantParams: []any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gl, _ := newTestGormLogger()
			give := append([]any(nil), tt.params...)
			gotSQL, gotParams := gl.ParamsFilter(
				context.Background(), tt.sql, give...,
			)

			// GORM may reuse the params slice for rebind or retry, so
			// the filter must never mutate the caller's values.
			for i := range give {
				if give[i] != tt.params[i] {
					t.Errorf("caller params[%d] mutated: got %#v, want %#v",
						i, give[i], tt.params[i])
				}
			}

			if gotSQL != tt.wantSQL {
				t.Errorf("SQL = %q, want %q", gotSQL, tt.wantSQL)
			}
			if len(gotParams) != len(tt.wantParams) {
				t.Fatalf("params length = %d, want %d", len(gotParams), len(tt.wantParams))
			}
			for i := range gotParams {
				got := gotParams[i]
				want := tt.wantParams[i]

				// Compare as strings for mockGormHider results
				gotStr, gotOk := got.(string)
				wantStr, wantOk := want.(string)
				if gotOk && wantOk {
					if gotStr != wantStr {
						t.Errorf("params[%d] = %q, want %q", i, gotStr, wantStr)
					}
					continue
				}

				if got != want {
					t.Errorf("params[%d] = %v, want %v", i, got, want)
				}
			}
		})
	}
}
