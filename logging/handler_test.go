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
	"fmt"
	"log/slog"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"gitlab.com/iglou.eu/goulc/logging/model"
)

func TestHandler_ConcurrentWritesNoBufferPollution(t *testing.T) {
	t.Run("concurrent_writes_no_buffer_pollution", func(t *testing.T) {
		const numGoroutines = 100
		const numIterations = 50

		var mu sync.Mutex
		var buf bytes.Buffer

		w := &model.Writer{
			Out: &lockedWriter{mu: &mu, buf: &buf},
			Err: &lockedWriter{mu: &mu, buf: &buf},
		}

		h := NewHandler(nil, w, &HandlerOptions{Config: model.Config{Colored: false}})

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := range numGoroutines {
			go func(id int) {
				defer wg.Done()
				for j := range numIterations {
					msg := fmt.Sprintf("goroutine-%d message-%d", id, j)
					r := slog.NewRecord(time.Now(), slog.LevelInfo, msg, 0)
					if err := h.Handle(context.Background(), r); err != nil {
						t.Errorf("Handle() returned error for goroutine %d iteration %d: %v", id, j, err)
					}
				}
			}(i)
		}

		wg.Wait()

		// Parse the captured output
		mu.Lock()
		output := buf.String()
		mu.Unlock()

		lines := strings.Split(strings.TrimSpace(output), "\n")

		expectedTotal := numGoroutines * numIterations
		if len(lines) != expectedTotal {
			t.Errorf("expected %d lines, got %d", expectedTotal, len(lines))
		}

		// Build set of expected messages
		expected := make(map[string]struct{}, expectedTotal)
		for i := 0; i < numGoroutines; i++ {
			for j := 0; j < numIterations; j++ {
				key := fmt.Sprintf("goroutine-%d message-%d", i, j)
				expected[key] = struct{}{}
			}
		}

		// Regex to extract the message portion from each log line.
		// Log format: "[YYYY-MM-DD HH:MM:SS] [LEVEL] <message>"
		msgRe := regexp.MustCompile(`^\[.*?\]\s+\[.*?\]\s+(.+)$`)

		seen := make(map[string]int, expectedTotal)

		for lineNum, line := range lines {
			matches := msgRe.FindStringSubmatch(line)
			if matches == nil {
				t.Errorf("line %d does not match expected log format: %q", lineNum+1, line)
				continue
			}

			msg := matches[1]

			// Verify the message is one we expected
			if _, ok := expected[msg]; !ok {
				t.Errorf("line %d contains unexpected message (possible buffer pollution): %q", lineNum+1, msg)
				continue
			}

			seen[msg]++

			// Check for cross-contamination: the message portion must not
			// contain fragments from another goroutine's identifier
			parts := strings.SplitN(msg, " ", 2)
			if len(parts) == 2 {
				goroutineID := parts[0] // e.g. "goroutine-42"
				// Verify no other goroutine-N pattern appears in the line
				// beyond the expected one
				otherGoroutineRe := regexp.MustCompile(`goroutine-\d+`)
				allMatches := otherGoroutineRe.FindAllString(line, -1)
				for _, m := range allMatches {
					if m != goroutineID {
						t.Errorf("line %d has cross-contamination: expected only %s but found %s in: %q",
							lineNum+1, goroutineID, m, line)
					}
				}
			}
		}

		// Verify every expected message appeared exactly once
		for key := range expected {
			count := seen[key]
			if count != 1 {
				t.Errorf("message %q appeared %d times, expected exactly 1", key, count)
			}
		}
	})
}

func TestNewHandler(t *testing.T) {
	tests := []struct {
		name    string
		opts    *HandlerOptions
		checkFn func(t *testing.T, h *Handler)
	}{
		{
			name: "nil_opts",
			opts: nil,
			checkFn: func(t *testing.T, h *Handler) {
				if h == nil {
					t.Fatal("NewHandler returned nil with nil opts")
				}
				if h.opt.TimeFormat != defaultTimeFormat {
					t.Errorf("expected default TimeFormat %q, got %q", defaultTimeFormat, h.opt.TimeFormat)
				}
			},
		},
		{
			name: "force_syslog_disables_colored",
			opts: &HandlerOptions{Config: model.Config{Colored: true, ForceSyslog: true}},
			checkFn: func(t *testing.T, h *Handler) {
				if h.opt.Colored {
					t.Error("expected Colored=false when ForceSyslog=true")
				}
				if !h.opt.ForceSyslog {
					t.Error("expected ForceSyslog=true")
				}
			},
		},
		{
			name: "basepath_preserved_as_is",
			opts: &HandlerOptions{
				Config:   model.Config{AddSource: true},
				BasePath: "/home/user/project/cmd/app",
			},
			checkFn: func(t *testing.T, h *Handler) {
				expected := "/home/user/project/cmd/app"
				if h.opt.BasePath != expected {
					t.Errorf("expected BasePath %q, got %q", expected, h.opt.BasePath)
				}
			},
		},
		{
			name: "empty_timeformat_defaults",
			opts: &HandlerOptions{Config: model.Config{TimeFormat: ""}},
			checkFn: func(t *testing.T, h *Handler) {
				if h.opt.TimeFormat != defaultTimeFormat {
					t.Errorf("expected default TimeFormat %q, got %q", defaultTimeFormat, h.opt.TimeFormat)
				}
			},
		},
		{
			name: "custom_timeformat_preserved",
			opts: &HandlerOptions{Config: model.Config{TimeFormat: "15:04"}},
			checkFn: func(t *testing.T, h *Handler) {
				if h.opt.TimeFormat != "15:04" {
					t.Errorf("expected TimeFormat %q, got %q", "15:04", h.opt.TimeFormat)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := &model.Writer{Out: &buf, Err: &buf}
			h := NewHandler(nil, w, tt.opts)
			tt.checkFn(t, h)
		})
	}
}

func TestHandler_Enabled(t *testing.T) {
	var buf bytes.Buffer
	w := &model.Writer{Out: &buf, Err: &buf}
	h := NewHandler(nil, w, &HandlerOptions{HandlerOptions: slog.HandlerOptions{Level: slog.LevelInfo}})

	tests := []struct {
		level    slog.Level
		expected bool
	}{
		{slog.LevelDebug, false},
		{slog.LevelInfo, true},
		{slog.LevelWarn, true},
		{slog.LevelError, true},
	}

	for _, tt := range tests {
		t.Run(tt.level.String(), func(t *testing.T) {
			got := h.Enabled(context.Background(), tt.level)
			if got != tt.expected {
				t.Errorf("Enabled(%s) = %v, want %v", tt.level, got, tt.expected)
			}
		})
	}
}

func TestHandler_WithAttrs(t *testing.T) {
	var buf bytes.Buffer
	w := &model.Writer{Out: &buf, Err: &buf}
	h := NewHandler(nil, w, &HandlerOptions{Config: model.Config{Colored: false}})

	attrs := []slog.Attr{
		slog.String("key1", "val1"),
		slog.Int("key2", 42),
	}
	h2 := h.WithAttrs(attrs)

	if h2 == slog.Handler(h) {
		t.Error("WithAttrs should return a different handler instance")
	}

	// Log a message through the new handler and check attrs appear
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "test message", 0)
	if err := h2.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "key1=val1") {
		t.Errorf("expected output to contain 'key1=val1', got %q", output)
	}
	if !strings.Contains(output, "key2=42") {
		t.Errorf("expected output to contain 'key2=42', got %q", output)
	}
}

func TestHandler_WithGroup(t *testing.T) {
	var buf bytes.Buffer
	w := &model.Writer{Out: &buf, Err: &buf}
	h := NewHandler(nil, w, &HandlerOptions{Config: model.Config{Colored: false}})

	h2 := h.WithGroup("mygroup")

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "grouped message", 0)
	if err := h2.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[G:mygroup]") {
		t.Errorf("expected output to contain '[G:mygroup]', got %q", output)
	}
}

func TestHandler_Handle(t *testing.T) {
	// Helper to get a valid PC for source tests
	var testPC uintptr
	func() {
		var pcs [1]uintptr
		runtime.Callers(1, pcs[:])
		testPC = pcs[0]
	}()

	tests := []struct {
		name     string
		opts     *HandlerOptions
		level    slog.Level
		msg      string
		attrs    []slog.Attr
		usePC    bool
		checkOut func(t *testing.T, out, errOut string)
	}{
		{
			name:  "basic_message_format",
			opts:  &HandlerOptions{Config: model.Config{Colored: false}},
			level: slog.LevelInfo,
			msg:   "hello world",
			checkOut: func(t *testing.T, out, _ string) {
				// Check timestamp pattern
				tsRe := regexp.MustCompile(`\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\]`)
				if !tsRe.MatchString(out) {
					t.Errorf("expected timestamp pattern in output, got %q", out)
				}
				if !strings.Contains(out, "[INFO]") {
					t.Errorf("expected [INFO] in output, got %q", out)
				}
				if !strings.Contains(out, "hello world") {
					t.Errorf("expected message in output, got %q", out)
				}
			},
		},
		{
			name:  "error_level_writes_to_err",
			opts:  &HandlerOptions{Config: model.Config{Colored: false}},
			level: slog.LevelError,
			msg:   "error message",
			checkOut: func(t *testing.T, out, errOut string) {
				if out != "" {
					t.Errorf("expected nothing on Out for ERROR, got %q", out)
				}
				if !strings.Contains(errOut, "error message") {
					t.Errorf("expected error message on Err, got %q", errOut)
				}
			},
		},
		{
			name:  "info_level_writes_to_out",
			opts:  &HandlerOptions{Config: model.Config{Colored: false}},
			level: slog.LevelInfo,
			msg:   "info message",
			checkOut: func(t *testing.T, out, errOut string) {
				if !strings.Contains(out, "info message") {
					t.Errorf("expected info message on Out, got %q", out)
				}
				if errOut != "" {
					t.Errorf("expected nothing on Err for INFO, got %q", errOut)
				}
			},
		},
		{
			name:  "with_add_source",
			opts:  &HandlerOptions{Config: model.Config{Colored: false, AddSource: true}},
			level: slog.LevelInfo,
			msg:   "sourced message",
			usePC: true,
			checkOut: func(t *testing.T, out, _ string) {
				// Source output should contain a file reference with colon and line number
				srcRe := regexp.MustCompile(`\S+\.go:\d+:`)
				if !srcRe.MatchString(out) {
					t.Errorf("expected source file reference in output, got %q", out)
				}
			},
		},
		{
			name:  "with_colored_output",
			opts:  &HandlerOptions{Config: model.Config{Colored: true}},
			level: slog.LevelInfo,
			msg:   "colored message",
			checkOut: func(t *testing.T, out, _ string) {
				if !strings.Contains(out, "\033[") {
					t.Errorf("expected ANSI escape codes in output, got %q", out)
				}
			},
		},
		{
			name:  "with_force_syslog",
			opts:  &HandlerOptions{Config: model.Config{ForceSyslog: true}},
			level: slog.LevelInfo,
			msg:   "syslog message",
			checkOut: func(t *testing.T, out, _ string) {
				expectedPrefix := BuildSyslogPrefix(slog.LevelInfo)
				if expectedPrefix == "" {
					t.Skip("syslog prefix is empty on this platform")
				}
				if !strings.HasPrefix(out, expectedPrefix) {
					t.Errorf("expected output to start with syslog prefix %q, got %q", expectedPrefix, out)
				}
			},
		},
		{
			name:  "with_record_attributes",
			opts:  &HandlerOptions{Config: model.Config{Colored: false}},
			level: slog.LevelInfo,
			msg:   "attrs message",
			attrs: []slog.Attr{slog.String("foo", "bar")},
			checkOut: func(t *testing.T, out, _ string) {
				if !strings.Contains(out, "\t- foo=bar") {
					t.Errorf("expected indented attribute line, got %q", out)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outBuf, errBuf bytes.Buffer
			w := &model.Writer{Out: &outBuf, Err: &errBuf}
			h := NewHandler(nil, w, tt.opts)

			var pc uintptr
			if tt.usePC {
				pc = testPC
			}

			r := slog.NewRecord(time.Now(), tt.level, tt.msg, pc)
			for _, a := range tt.attrs {
				r.AddAttrs(a)
			}

			if err := h.Handle(context.Background(), r); err != nil {
				t.Fatalf("Handle() error: %v", err)
			}

			tt.checkOut(t, outBuf.String(), errBuf.String())
		})
	}
}

func TestHandler_Cancel(t *testing.T) {
	t.Run("with_cancel_func", func(t *testing.T) {
		called := false
		cancelFn := func() { called = true }

		var buf bytes.Buffer
		w := &model.Writer{Out: &buf, Err: &buf}
		h := NewHandler(cancelFn, w, nil)

		result := h.Cancel()
		if !result {
			t.Error("Cancel() should return true when cancel func is set")
		}
		if !called {
			t.Error("expected cancel function to be called")
		}
	})

	t.Run("nil_cancel_func", func(t *testing.T) {
		var buf bytes.Buffer
		w := &model.Writer{Out: &buf, Err: &buf}
		h := NewHandler(nil, w, nil)

		result := h.Cancel()
		if result {
			t.Error("Cancel() should return false when cancel func is nil")
		}
	})
}

func TestHandler_colorize(t *testing.T) {
	tests := []struct {
		name     string
		colored  bool
		input    string
		contains string
		noAnsi   bool
	}{
		{
			name:     "colored_wraps_with_ansi",
			colored:  true,
			input:    "hello",
			contains: "\033[",
		},
		{
			name:    "not_colored_writes_as_is",
			colored: false,
			input:   "hello",
			noAnsi:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outBuf bytes.Buffer
			w := &model.Writer{Out: &outBuf, Err: &outBuf}
			h := NewHandler(nil, w, &HandlerOptions{Config: model.Config{Colored: tt.colored}})

			var buf bytes.Buffer
			h.colorize(&buf, colorCyan, tt.input)
			got := buf.String()

			if tt.noAnsi {
				if strings.Contains(got, "\033[") {
					t.Errorf("expected no ANSI codes, got %q", got)
				}
				if got != tt.input {
					t.Errorf("expected %q, got %q", tt.input, got)
				}
			} else {
				if !strings.Contains(got, tt.contains) {
					t.Errorf("expected ANSI code in output, got %q", got)
				}
				if !strings.Contains(got, tt.input) {
					t.Errorf("expected input string in output, got %q", got)
				}
				if !strings.Contains(got, string(colorReset)) {
					t.Errorf("expected color reset in output, got %q", got)
				}
			}
		})
	}
}

func TestHandler_writeSyslogPrefix(t *testing.T) {
	t.Run("force_syslog_true", func(t *testing.T) {
		var outBuf bytes.Buffer
		w := &model.Writer{Out: &outBuf, Err: &outBuf}
		h := NewHandler(nil, w, &HandlerOptions{Config: model.Config{ForceSyslog: true}})

		var buf bytes.Buffer
		h.writeSyslogPrefix(&buf, slog.LevelInfo)

		got := buf.String()
		expected := BuildSyslogPrefix(slog.LevelInfo)
		if expected == "" {
			t.Skip("syslog prefix is empty on this platform")
		}
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	})

	t.Run("force_syslog_false", func(t *testing.T) {
		var outBuf bytes.Buffer
		w := &model.Writer{Out: &outBuf, Err: &outBuf}
		h := NewHandler(nil, w, &HandlerOptions{Config: model.Config{ForceSyslog: false}})

		var buf bytes.Buffer
		h.writeSyslogPrefix(&buf, slog.LevelInfo)

		if buf.Len() != 0 {
			t.Errorf("expected empty buffer when ForceSyslog=false, got %q", buf.String())
		}
	})
}

func TestHandler_AttributeLines(t *testing.T) {
	t.Run("basic_attribute", func(t *testing.T) {
		var outBuf bytes.Buffer
		w := &model.Writer{Out: &outBuf, Err: &outBuf}
		h := NewHandler(nil, w, &HandlerOptions{Config: model.Config{Colored: false}})

		r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
		r.AddAttrs(slog.String("key", "value"))
		if err := h.Handle(context.Background(), r); err != nil {
			t.Fatalf("Handle() error: %v", err)
		}

		got := outBuf.String()
		if !strings.Contains(got, "\n\t- key=value") {
			t.Errorf("expected newline-tab-dash attribute line, got %q", got)
		}
	})

	t.Run("with_syslog_prefix", func(t *testing.T) {
		var outBuf bytes.Buffer
		w := &model.Writer{Out: &outBuf, Err: &outBuf}
		h := NewHandler(nil, w, &HandlerOptions{Config: model.Config{ForceSyslog: true, Colored: false}})

		r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
		r.AddAttrs(slog.String("key", "value"))
		if err := h.Handle(context.Background(), r); err != nil {
			t.Fatalf("Handle() error: %v", err)
		}

		prefix := BuildSyslogPrefix(slog.LevelInfo)
		if prefix == "" {
			t.Skip("syslog prefix is empty on this platform")
		}

		// Every line of a record must carry the prefix, so multi-line
		// records stay consistent for the journal.
		got := strings.TrimSuffix(outBuf.String(), "\n")
		for i, line := range strings.Split(got, "\n") {
			if !strings.HasPrefix(line, prefix) {
				t.Errorf("line %d misses syslog prefix %q: %q", i, prefix, line)
			}
		}
	})
}

func TestHandler_WithGroupAndAttrs(t *testing.T) {
	t.Run("group_attrs_group", func(t *testing.T) {
		var buf bytes.Buffer
		w := &model.Writer{Out: &buf, Err: &buf}
		h := NewHandler(nil, w, &HandlerOptions{Config: model.Config{Colored: false}})

		attrs := []slog.Attr{slog.String("k", "v")}
		h2 := h.WithGroup("a").WithAttrs(attrs).WithGroup("b")

		r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
		if err := h2.Handle(context.Background(), r); err != nil {
			t.Fatalf("Handle() error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "[G:a.b]") {
			t.Errorf("expected output to contain '[G:a.b]', got %q", output)
		}
		if !strings.Contains(output, "a.k=v") {
			t.Errorf("expected output to contain 'a.k=v', got %q", output)
		}
	})

	t.Run("attrs_then_group", func(t *testing.T) {
		var buf bytes.Buffer
		w := &model.Writer{Out: &buf, Err: &buf}
		h := NewHandler(nil, w, &HandlerOptions{Config: model.Config{Colored: false}})

		attrs := []slog.Attr{slog.String("x", "y")}
		h2 := h.WithAttrs(attrs).WithGroup("g")

		r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
		if err := h2.Handle(context.Background(), r); err != nil {
			t.Fatalf("Handle() error: %v", err)
		}

		output := buf.String()
		if !strings.Contains(output, "[G:g]") {
			t.Errorf("expected output to contain '[G:g]', got %q", output)
		}
		if !strings.Contains(output, "x=y") {
			t.Errorf("expected output to contain 'x=y', got %q", output)
		}
	})
}

// lockedWriter is a thread-safe writer that appends to a shared bytes.Buffer.
type lockedWriter struct {
	mu  *sync.Mutex
	buf *bytes.Buffer
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buf.Write(p)
}
