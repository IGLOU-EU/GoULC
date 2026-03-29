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
	"os"
	"os/exec"
	"testing"

	"gitlab.com/iglou.eu/goulc/logging/model"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *model.Config
		wantErr bool
	}{
		{
			name:    "valid_config",
			cfg:     &model.Config{Level: "INFO"},
			wantErr: false,
		},
		{
			name:    "invalid_level",
			cfg:     &model.Config{Level: "INVALID"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := New("", tt.cfg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if l == nil {
				t.Fatal("expected logger, got nil")
			}
		})
	}
}

func TestNewWithWriter(t *testing.T) {
	tests := []struct {
		name    string
		writer  *model.Writer
		cfg     *model.Config
		wantErr error
	}{
		{
			name:   "nil_writer_uses_default",
			writer: nil,
			cfg:    &model.Config{Level: "INFO"},
		},
		{
			name: "custom_writer_out_and_err",
			writer: &model.Writer{
				Out: &bytes.Buffer{},
				Err: &bytes.Buffer{},
			},
			cfg: &model.Config{Level: "DEBUG"},
		},
		{
			name: "err_nil_defaults_to_out",
			writer: &model.Writer{
				Out: &bytes.Buffer{},
				Err: nil,
			},
			cfg: &model.Config{Level: "WARN"},
		},
		{
			name: "out_nil_returns_error",
			writer: &model.Writer{
				Out: nil,
				Err: &bytes.Buffer{},
			},
			wantErr: ErrWriterOutNil,
		},
		{
			name:   "nil_cfg_uses_default",
			writer: &model.Writer{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}},
			cfg:    nil,
		},
		{
			name:    "invalid_level_string",
			writer:  &model.Writer{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}},
			cfg:     &model.Config{Level: "TRACE"},
			wantErr: ErrLogLevelUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := NewWithWriter("", tt.writer, tt.cfg)
			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error %v, got %v", tt.wantErr, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if l == nil {
				t.Fatal("expected logger, got nil")
			}
		})
	}
}

func TestCritical(t *testing.T) {
	t.Run("cancel_called", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		buf := &bytes.Buffer{}
		w := &model.Writer{Out: buf, Err: buf}
		cfg := &model.Config{
			Level:  "DEBUG",
			Cancel: cancel,
		}

		l, err := NewWithWriter("", w, cfg)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		Critical(l, "test critical")

		select {
		case <-ctx.Done():
			// Context was cancelled as expected
		default:
			t.Fatal("expected context to be cancelled after Critical call")
		}
	})

	t.Run("non_handler_does_not_panic", func(t *testing.T) {
		if os.Getenv("TEST_CRITICAL_EXIT") == "1" {
			buf := &bytes.Buffer{}
			l := slog.New(slog.NewTextHandler(buf, nil))
			Critical(l, "test exit")
			return
		}
		cmd := exec.Command(os.Args[0], "-test.run=TestCritical/non_handler_does_not_panic")
		cmd.Env = append(os.Environ(), "TEST_CRITICAL_EXIT=1")
		err := cmd.Run()
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
			t.Fatalf("expected exit code 1, got: %v", err)
		}
	})
}

func TestGetLevel(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    slog.Level
		wantErr bool
	}{
		{
			name:  "debug",
			input: "DEBUG",
			want:  slog.LevelDebug,
		},
		{
			name:  "info",
			input: "INFO",
			want:  slog.LevelInfo,
		},
		{
			name:  "warn",
			input: "WARN",
			want:  slog.LevelWarn,
		},
		{
			name:  "error",
			input: "ERROR",
			want:  slog.LevelError,
		},
		{
			name:    "invalid",
			input:   "INVALID",
			wantErr: true,
		},
		{
			name:    "empty",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getLevel(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !errors.Is(err, ErrLogLevelUnknown) {
					t.Fatalf("expected error %v, got %v",
						ErrLogLevelUnknown, err)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
