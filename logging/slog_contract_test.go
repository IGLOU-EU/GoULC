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
	"log/slog"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/slogtest"
	"time"

	"gitlab.com/iglou.eu/goulc/logging/model"
)

// TestHandler_ConcurrentDerivedHandlers verifies that a handler and its
// WithAttrs/WithGroup clones serialize writes to the shared writer. The
// writer is deliberately not synchronized: the handler mutex is the only
// protection, so -race fails if clones do not share it.
func TestHandler_ConcurrentDerivedHandlers(t *testing.T) {
	const perHandler = 100

	var buf bytes.Buffer
	w := &model.Writer{Out: &buf, Err: &buf}
	h := NewHandler(nil, w, &HandlerOptions{Config: model.Config{Colored: false}})

	handlers := []slog.Handler{
		h,
		h.WithAttrs([]slog.Attr{slog.String("a", "b")}),
		h.WithGroup("grp"),
		h.WithGroup("grp").WithAttrs([]slog.Attr{slog.String("c", "d")}),
	}

	var wg sync.WaitGroup
	for i, hh := range handlers {
		wg.Add(1)
		go func(id int, hh slog.Handler) {
			defer wg.Done()
			for j := range perHandler {
				r := slog.NewRecord(time.Now(), slog.LevelInfo, "concurrent", 0)
				r.AddAttrs(slog.Int("id", id), slog.Int("iter", j))
				if err := hh.Handle(context.Background(), r); err != nil {
					t.Errorf("Handle() error: %v", err)
				}
			}
		}(i, hh)
	}
	wg.Wait()

	headerRe := regexp.MustCompile(`(?m)^\[\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\] \[INFO\]`)
	got := len(headerRe.FindAllString(buf.String(), -1))
	want := len(handlers) * perHandler
	if got != want {
		t.Errorf("expected %d records, got %d", want, got)
	}
}

func TestHandler_ReplaceAttrRedaction(t *testing.T) {
	redact := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == "password" {
			return slog.String("password", "[REDACTED]")
		}
		return a
	}

	tests := []struct {
		name  string
		logFn func(l *slog.Logger)
	}{
		{
			name:  "record_attr",
			logFn: func(l *slog.Logger) { l.Info("login", "password", "hunter2") },
		},
		{
			name:  "with_attrs",
			logFn: func(l *slog.Logger) { l.With("password", "hunter2").Info("login") },
		},
		{
			name: "grouped_attr",
			logFn: func(l *slog.Logger) {
				l.Info("login", slog.Group("auth", slog.String("password", "hunter2")))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := &model.Writer{Out: &buf, Err: &buf}
			h := NewHandler(nil, w, &HandlerOptions{
				HandlerOptions: slog.HandlerOptions{ReplaceAttr: redact},
			})

			tt.logFn(slog.New(h))

			out := buf.String()
			if strings.Contains(out, "hunter2") {
				t.Errorf("secret leaked despite ReplaceAttr, output: %q", out)
			}
			if !strings.Contains(out, "[REDACTED]") {
				t.Errorf("expected redacted marker in output, got %q", out)
			}
		})
	}
}

func TestHandler_ReplaceAttrDiscard(t *testing.T) {
	discard := func(_ []string, a slog.Attr) slog.Attr {
		if a.Key == "password" {
			return slog.Attr{}
		}
		return a
	}

	var buf bytes.Buffer
	w := &model.Writer{Out: &buf, Err: &buf}
	h := NewHandler(nil, w, &HandlerOptions{
		HandlerOptions: slog.HandlerOptions{ReplaceAttr: discard},
	})

	slog.New(h).Info("login", "password", "hunter2", "user", "john")

	out := buf.String()
	if strings.Contains(out, "password") || strings.Contains(out, "hunter2") {
		t.Errorf("discarded attribute still present, output: %q", out)
	}
	if !strings.Contains(out, "user=john") {
		t.Errorf("expected remaining attribute in output, got %q", out)
	}
}

func TestHandler_ReplaceAttrContract(t *testing.T) {
	var gotGroups []string
	var gotKind slog.Kind

	rep := func(groups []string, a slog.Attr) slog.Attr {
		if a.Key == "token" {
			gotGroups = append([]string(nil), groups...)
			gotKind = a.Value.Kind()
		}
		return a
	}

	var buf bytes.Buffer
	w := &model.Writer{Out: &buf, Err: &buf}
	h := NewHandler(nil, w, &HandlerOptions{
		HandlerOptions: slog.HandlerOptions{ReplaceAttr: rep},
	})

	slog.New(h).WithGroup("s").Info("msg",
		slog.Group("g", slog.Any("token", testSecret{raw: "raw-secret"})))

	wantGroups := []string{"s", "g"}
	if !reflect.DeepEqual(gotGroups, wantGroups) {
		t.Errorf("ReplaceAttr groups = %v, want %v", gotGroups, wantGroups)
	}
	if gotKind != slog.KindString {
		t.Errorf("ReplaceAttr got kind %v, want resolved %v",
			gotKind, slog.KindString)
	}
}

// testSecret is a slog.LogValuer whose raw content must never be logged.
type testSecret struct {
	raw string
}

func (s testSecret) LogValue() slog.Value { return slog.StringValue("***") }

func TestHandler_LogValuerResolve(t *testing.T) {
	tests := []struct {
		name  string
		logFn func(l *slog.Logger)
	}{
		{
			name: "record_attr",
			logFn: func(l *slog.Logger) {
				l.Info("msg", "token", testSecret{raw: "raw-secret"})
			},
		},
		{
			name: "with_attrs",
			logFn: func(l *slog.Logger) {
				l.With("token", testSecret{raw: "raw-secret"}).Info("msg")
			},
		},
		{
			name: "grouped_record_attr",
			logFn: func(l *slog.Logger) {
				l.Info("msg", slog.Group("g",
					slog.Any("token", testSecret{raw: "raw-secret"})))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := &model.Writer{Out: &buf, Err: &buf}
			h := NewHandler(nil, w, nil)

			tt.logFn(slog.New(h))

			out := buf.String()
			if strings.Contains(out, "raw-secret") {
				t.Errorf("LogValuer raw value leaked, output: %q", out)
			}
			if !strings.Contains(out, "token=***") {
				t.Errorf("expected resolved value in output, got %q", out)
			}
		})
	}
}

func TestHandler_ControlCharEscaping(t *testing.T) {
	tests := []struct {
		name      string
		msg       string
		attrs     []slog.Attr
		wantLines int
		want      []string
		absent    []string
	}{
		{
			name:      "newline_in_value",
			msg:       "msg",
			attrs:     []slog.Attr{slog.String("k", "v\n[INFO] forged line")},
			wantLines: 2,
			want:      []string{`k="v\n[INFO] forged line"`},
			absent:    []string{"v\n[INFO]"},
		},
		{
			name:      "ansi_escape_in_value",
			msg:       "msg",
			attrs:     []slog.Attr{slog.String("k", "\x1b[31mred")},
			wantLines: 2,
			want:      []string{`k="\x1b[31mred"`},
			absent:    []string{"\x1b"},
		},
		{
			name:      "del_byte_in_value",
			msg:       "msg",
			attrs:     []slog.Attr{slog.String("k", "a\x7fb")},
			wantLines: 2,
			want:      []string{`k="a\x7fb"`},
			absent:    []string{"\x7f"},
		},
		{
			name:      "newline_in_message",
			msg:       "bad\nmessage",
			wantLines: 1,
			want:      []string{`"bad\nmessage"`},
			absent:    []string{"bad\nmessage"},
		},
		{
			name:      "newline_in_key",
			msg:       "msg",
			attrs:     []slog.Attr{slog.String("bad\nkey", "v")},
			wantLines: 2,
			want:      []string{`"bad\nkey"`},
			absent:    []string{"bad\nkey"},
		},
		{
			name:      "plain_value_not_quoted",
			msg:       "msg",
			attrs:     []slog.Attr{slog.String("k", "v")},
			wantLines: 2,
			want:      []string{"k=v"},
			absent:    []string{`k="v"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := &model.Writer{Out: &buf, Err: &buf}
			h := NewHandler(nil, w, nil)

			r := slog.NewRecord(time.Now(), slog.LevelInfo, tt.msg, 0)
			r.AddAttrs(tt.attrs...)
			if err := h.Handle(context.Background(), r); err != nil {
				t.Fatalf("Handle() error: %v", err)
			}

			out := buf.String()
			lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
			if len(lines) != tt.wantLines {
				t.Errorf("expected %d lines, got %d: %q",
					tt.wantLines, len(lines), out)
			}
			for _, s := range tt.want {
				if !strings.Contains(out, s) {
					t.Errorf("expected %q in output, got %q", s, out)
				}
			}
			for _, s := range tt.absent {
				if strings.Contains(out, s) {
					t.Errorf("did not expect %q in output, got %q", s, out)
				}
			}
		})
	}
}

func TestHandler_ValueKinds(t *testing.T) {
	giveTime := time.Date(2026, 7, 25, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		give slog.Attr
		want string
	}{
		{"string", slog.String("k", "v"), "k=v"},
		{"int64", slog.Int64("k", -42), "k=-42"},
		{"uint64", slog.Uint64("k", 42), "k=42"},
		{"float64", slog.Float64("k", 1.5), "k=1.5"},
		{"bool", slog.Bool("k", true), "k=true"},
		{"duration", slog.Duration("k", 1500*time.Millisecond), "k=1.5s"},
		{"time", slog.Time("k", giveTime), "k=2026-07-25T10:30:00Z"},
		{"any", slog.Any("k", struct{ A int }{7}), "k={7}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			w := &model.Writer{Out: &buf, Err: &buf}
			h := NewHandler(nil, w, nil)

			r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
			r.AddAttrs(tt.give)
			if err := h.Handle(context.Background(), r); err != nil {
				t.Fatalf("Handle() error: %v", err)
			}

			want := "\t- " + tt.want + "\n"
			if !strings.Contains(buf.String(), want) {
				t.Errorf("expected %q in output, got %q", want, buf.String())
			}
		})
	}
}

func TestHandler_WithAttrsEmptyReturnsReceiver(t *testing.T) {
	var buf bytes.Buffer
	w := &model.Writer{Out: &buf, Err: &buf}
	h := NewHandler(nil, w, nil)

	if h.WithAttrs(nil) != slog.Handler(h) {
		t.Error("WithAttrs(nil) should return the receiver")
	}
}

// TestHandler_WithAttrsEmptyGroupElided exercises the handler directly:
// slog.Record elides empty groups on its own, but WithAttrs receives
// them untouched and must drop them too.
func TestHandler_WithAttrsEmptyGroupElided(t *testing.T) {
	var buf bytes.Buffer
	w := &model.Writer{Out: &buf, Err: &buf}
	h := NewHandler(nil, w, nil)

	h2 := h.WithAttrs([]slog.Attr{slog.Group("G")})

	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
	if err := h2.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "G") && strings.Contains(out, "\t- ") {
		t.Errorf("empty group must be elided, got %q", out)
	}
	if lines := strings.Count(out, "\n"); lines != 1 {
		t.Errorf("expected a single line record, got %q", out)
	}
}

func TestHandler_UnknownSourceFallback(t *testing.T) {
	var buf bytes.Buffer
	w := &model.Writer{Out: &buf, Err: &buf}
	h := NewHandler(nil, w, &HandlerOptions{
		Config: model.Config{AddSource: true},
	})

	// A non-zero PC that maps to no known function must not drop the
	// source marker silently.
	r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 1)
	if err := h.Handle(context.Background(), r); err != nil {
		t.Fatalf("Handle() error: %v", err)
	}

	if !strings.Contains(buf.String(), "???:") {
		t.Errorf("expected ??? source fallback, got %q", buf.String())
	}
}

// TestHandler_HostileGroupNameEscaped covers the [G:...] marker: a group
// name is attacker-influenced data like any attribute, so control bytes
// must never reach the output through it either.
func TestHandler_HostileGroupNameEscaped(t *testing.T) {
	const hostile = "grp\r\n<6>injected \x1b[31mred"

	t.Run("marker_and_keys_quoted", func(t *testing.T) {
		var buf bytes.Buffer
		w := &model.Writer{Out: &buf, Err: &buf}
		h := NewHandler(nil, w, nil).WithGroup(hostile)

		r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
		r.AddAttrs(slog.String("k", "v"))
		if err := h.Handle(context.Background(), r); err != nil {
			t.Fatalf("Handle() error: %v", err)
		}

		out := buf.String()
		for i := 0; i < len(out); i++ {
			c := out[i]
			// The line separator and the attribute indent are the only
			// control bytes the format itself emits.
			if c == '\n' || c == '\t' {
				continue
			}
			if c < 0x20 || c == 0x7f {
				t.Fatalf("raw control byte %#02x at offset %d in %q", c, i, out)
			}
		}

		want := `[G:"grp\r\n<6>injected \x1b[31mred"]`
		if !strings.Contains(out, want) {
			t.Errorf("expected quoted group marker %q, got %q", want, out)
		}
	})

	t.Run("replace_attr_sees_raw_name", func(t *testing.T) {
		var gotGroups []string
		rep := func(groups []string, a slog.Attr) slog.Attr {
			gotGroups = append([]string(nil), groups...)
			return a
		}

		var buf bytes.Buffer
		w := &model.Writer{Out: &buf, Err: &buf}
		h := NewHandler(nil, w, &HandlerOptions{
			HandlerOptions: slog.HandlerOptions{ReplaceAttr: rep},
		}).WithGroup(hostile)

		r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
		r.AddAttrs(slog.String("k", "v"))
		if err := h.Handle(context.Background(), r); err != nil {
			t.Fatalf("Handle() error: %v", err)
		}

		want := []string{hostile}
		if !reflect.DeepEqual(gotGroups, want) {
			t.Errorf("ReplaceAttr groups = %q, want raw %q", gotGroups, want)
		}
	})
}

func TestHandler_ErrorLevelRouting(t *testing.T) {
	tests := []struct {
		name    string
		level   slog.Level
		wantErr bool
	}{
		{"info_to_out", slog.LevelInfo, false},
		{"warn_to_out", slog.LevelWarn, false},
		{"error_to_err", slog.LevelError, true},
		{"above_error_to_err", slog.LevelError + 4, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outBuf, errBuf bytes.Buffer
			w := &model.Writer{Out: &outBuf, Err: &errBuf}
			h := NewHandler(nil, w, nil)

			r := slog.NewRecord(time.Now(), tt.level, "routing", 0)
			if err := h.Handle(context.Background(), r); err != nil {
				t.Fatalf("Handle() error: %v", err)
			}

			if tt.wantErr {
				if errBuf.Len() == 0 || outBuf.Len() != 0 {
					t.Errorf("level %v: expected record on Err only, "+
						"out=%q err=%q", tt.level, outBuf.String(), errBuf.String())
				}
				return
			}
			if outBuf.Len() == 0 || errBuf.Len() != 0 {
				t.Errorf("level %v: expected record on Out only, out=%q err=%q",
					tt.level, outBuf.String(), errBuf.String())
			}
		})
	}
}

func TestHandler_WithGroupContract(t *testing.T) {
	var buf bytes.Buffer
	w := &model.Writer{Out: &buf, Err: &buf}
	h := NewHandler(nil, w, nil)

	t.Run("empty_name_returns_receiver", func(t *testing.T) {
		if h.WithGroup("") != slog.Handler(h) {
			t.Error("WithGroup(\"\") should return the receiver")
		}
	})

	t.Run("no_leading_dot", func(t *testing.T) {
		buf.Reset()
		r := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)
		if err := h.WithGroup("grp").Handle(context.Background(), r); err != nil {
			t.Fatalf("Handle() error: %v", err)
		}
		out := buf.String()
		if !strings.Contains(out, "[G:grp]") {
			t.Errorf("expected group marker [G:grp], got %q", out)
		}
		if strings.Contains(out, "[G:.grp]") {
			t.Errorf("group marker has a leading dot: %q", out)
		}
	})
}

// TestHandler_SlogtestConformance runs the stdlib slog.Handler test suite
// against the handler, with a parser translating the text format back to
// the attribute maps slogtest expects. Assumed format deviations are
// documented in the package comment.
func TestHandler_SlogtestConformance(t *testing.T) {
	var buf bytes.Buffer
	w := &model.Writer{Out: &buf, Err: &buf}
	h := NewHandler(nil, w, &HandlerOptions{
		Config: model.Config{AddSource: true},
	})

	err := slogtest.TestHandler(h, func() []map[string]any {
		return parseLogOutput(t, buf.String())
	})
	if err != nil {
		t.Error(err)
	}
}

// logHeaderRe matches a record header line: optional timestamp, level,
// optional group marker, optional source reference, then the message.
var logHeaderRe = regexp.MustCompile(
	`^(?:\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\] )?` +
		`\[([A-Z]+(?:[+-]\d+)?)\] ` +
		`(?:\[G:[^\]]*\] )?` +
		`(?:(\S+\.go:\d+): )?` +
		`(.*)$`)

// parseLogOutput converts the handler text output into one attribute map
// per record, nesting dot-qualified keys, for slogtest.TestHandler.
func parseLogOutput(t *testing.T, out string) []map[string]any {
	t.Helper()

	var records []map[string]any
	var cur map[string]any

	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}

		if attr, ok := strings.CutPrefix(line, "\t- "); ok {
			if cur == nil {
				t.Fatalf("attribute line before any record header: %q", line)
			}
			key, val, found := strings.Cut(attr, "=")
			if !found {
				t.Fatalf("malformed attribute line: %q", line)
			}
			insertDotted(cur, key, unquoteIfNeeded(t, val))
			continue
		}

		m := logHeaderRe.FindStringSubmatch(line)
		if m == nil {
			t.Fatalf("malformed record header: %q", line)
		}

		cur = map[string]any{}
		if m[1] != "" {
			cur[slog.TimeKey] = m[1]
		}
		cur[slog.LevelKey] = m[2]
		if m[3] != "" {
			cur[slog.SourceKey] = m[3]
		}
		cur[slog.MessageKey] = unquoteIfNeeded(t, m[4])
		records = append(records, cur)
	}

	return records
}

// insertDotted stores val under a dot-qualified key, creating one nested
// map per group segment as slogtest expects.
func insertDotted(m map[string]any, key string, val any) {
	parts := strings.Split(key, ".")
	for _, p := range parts[:len(parts)-1] {
		sub, ok := m[p].(map[string]any)
		if !ok {
			sub = map[string]any{}
			m[p] = sub
		}
		m = sub
	}
	m[parts[len(parts)-1]] = val
}

// unquoteIfNeeded reverses the control-byte quoting applied on emission.
func unquoteIfNeeded(t *testing.T, s string) string {
	t.Helper()

	if !strings.HasPrefix(s, `"`) {
		return s
	}
	u, err := strconv.Unquote(s)
	if err != nil {
		t.Fatalf("cannot unquote %q: %v", s, err)
	}
	return u
}
