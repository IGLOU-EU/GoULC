/*
 * Copyright 2024 Adrien Kara
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
	"strconv"
	"sync"
	"time"

	"gitlab.com/iglou.eu/goulc/logging/model"
)

const (
	// bufferSize is the initial allocation size in bytes for the log
	// output buffer, chosen to minimize reallocations for typical log lines.
	bufferSize = 1024

	// maxBufferSize is the maximum capacity a pooled buffer is allowed to
	// have before being discarded instead of returned to the pool. This
	// prevents abnormally large log lines from permanently inflating memory.
	maxBufferSize = 4096

	// defaultTimeFormat is the fallback time layout used when no custom
	// TimeFormat is specified in HandlerOptions.
	defaultTimeFormat = "[2006-01-02 15:04:05]"
)

// bufPool is a pool of reusable bytes.Buffer to avoid allocating a new
// buffer for every log line. Buffers exceeding maxBufferSize are not
// returned to the pool.
var bufPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 0, bufferSize))
	},
}

// Handler implements slog.Handler interface with additional features:
// - Colored output support with ANSI colors
// - Separate writers for normal and error logs
// - Concurrent-safe logging with mutex protection
// - Source code location with customizable base path
type Handler struct {
	w   model.Writer
	opt HandlerOptions

	// groups holds the open WithGroup names, passed to ReplaceAttr and
	// rendered both as the "[G:...]" marker and as the key prefix.
	groups            []string
	prefix            string
	preformattedAttrs []string

	cancel context.CancelFunc

	// mu is held by pointer so every handler derived through
	// WithAttrs/WithGroup shares the same lock, serializing writes to
	// the shared writer (same pattern as the stdlib slog handlers).
	mu *sync.Mutex
}

var _ slog.Handler = (*Handler)(nil)

// HandlerOptions configures the behavior of the Handler.
type HandlerOptions struct {
	model.Config
	slog.HandlerOptions

	// BasePath is used to trim the full file path in source code references
	// For example, if BasePath is "/home/user/project", a source file
	// "/home/user/project/pkg/file.go" will be shown as "pkg/file.go"
	BasePath string
}

// NewHandler creates a new Handler with the given writer and options.
//   - The writer specifies where to write logs
//     (separate streams for normal and error logs).
//   - The opt parameter configures coloring, source code info, and base path.
//
// If opt is nil, default options will be used.
// The handler is concurrent-safe and implements the slog.Handler interface.
func NewHandler(
	cancel context.CancelFunc, w *model.Writer, opt *HandlerOptions,
) *Handler {
	var localOpt HandlerOptions
	if opt != nil {
		localOpt = *opt
	}

	if localOpt.ForceSyslog {
		localOpt.Colored = false
	}

	if localOpt.TimeFormat == "" {
		localOpt.TimeFormat = defaultTimeFormat
	}

	return &Handler{
		w:   *w,
		opt: localOpt,

		cancel: cancel,
		mu:     &sync.Mutex{},
	}
}

// Enabled implements slog.Handler interface and determines if a log level
// should be processed based on the handler's configured minimum level.
// A nil level means slog.LevelInfo, as for the stdlib handlers.
func (h *Handler) Enabled(_ context.Context, level slog.Level) bool {
	minLevel := slog.LevelInfo
	if h.opt.HandlerOptions.Level != nil {
		minLevel = h.opt.HandlerOptions.Level.Level()
	}

	return level >= minLevel
}

// WithAttrs implements slog.Handler interface and returns a new Handler
// that always logs the given attributes. The attributes are resolved,
// passed through ReplaceAttr, expanded from groups, and preformatted once
// so records only copy ready-made strings.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}

	newHandler := h.clone()

	buf := bufPool.Get().(*bytes.Buffer)
	for _, attr := range attrs {
		h.walkAttr(h.groups, h.prefix, attr, func(prefix string, a slog.Attr) {
			buf.Reset()
			appendKeyValue(buf, prefix, a)
			newHandler.preformattedAttrs = append(
				newHandler.preformattedAttrs, buf.String())
		})
	}
	if buf.Cap() <= maxBufferSize {
		bufPool.Put(buf)
	}

	return newHandler
}

// WithGroup returns a new handler with the given group name appended.
// The group qualifies the keys of subsequent attributes with a
// dot-separated prefix and is echoed as a "[G:...]" marker on each line.
func (h *Handler) WithGroup(group string) slog.Handler {
	if group == "" {
		return h
	}

	newHandler := h.clone()
	// groups keeps the raw name for ReplaceAttr, while the rendered
	// prefix is escaped once here so the [G:...] marker and the key
	// prefixes never carry control bytes, at no per-record cost.
	newHandler.groups = append(newHandler.groups, group)
	newHandler.prefix = h.prefix + escapeGroupName(group) + "."

	return newHandler
}

// clone copies the handler for WithAttrs/WithGroup derivation. Slices are
// copied so derived handlers never share append storage, while the mutex
// pointer is deliberately shared to keep writes serialized.
func (h *Handler) clone() *Handler {
	return &Handler{
		w:   h.w,
		opt: h.opt,

		groups:            append([]string(nil), h.groups...),
		prefix:            h.prefix,
		preformattedAttrs: append([]string(nil), h.preformattedAttrs...),

		cancel: h.cancel,
		mu:     h.mu,
	}
}

// Handle implements slog.Handler interface. It formats and writes a log record.
// The implementation:
// - Applies ANSI colors if enabled
// - Uses different writers for error and non-error logs
// - Includes timestamp, level, source location (if enabled), and message
// - Resolves, rewrites (ReplaceAttr), escapes, and writes all attributes
// - Is concurrent-safe through mutex protection
func (h *Handler) Handle(_ context.Context, r slog.Record) error {
	// Set colors if enabled
	var colorLevel color
	if h.opt.Colored {
		colorLevel = setColorLevel(r.Level)
	}

	// Get a buffer from the pool
	buf := bufPool.Get().(*bytes.Buffer)
	buf.Reset()

	// Add syslog prefix
	h.writeSyslogPrefix(buf, r.Level)

	// Date and time, skipped when zero per the slog.Handler contract
	if !r.Time.IsZero() {
		buf.Write(r.Time.AppendFormat(
			buf.AvailableBuffer(), h.opt.TimeFormat))
		buf.WriteByte(' ')
	}

	// Level
	buf.WriteByte('[')
	h.colorize(buf, colorLevel, r.Level.String())
	buf.WriteByte(']')
	buf.WriteByte(' ')

	// Group marker
	if h.prefix != "" {
		buf.WriteString("[G:")
		h.colorize(buf, colorLevel, h.prefix[:len(h.prefix)-1])
		buf.WriteByte(']')
		buf.WriteByte(' ')
	}

	// Source, skipped when the PC is zero per the slog.Handler contract
	if h.opt.Config.AddSource && r.PC != 0 {
		h.writeSource(buf, r.PC)
	}

	// Message
	appendEscaped(buf, r.Message)

	// Write preformatted attributes
	for _, s := range h.preformattedAttrs {
		h.writeAttrLineStart(buf, r.Level)
		buf.WriteString(s)
		h.colorEnd(buf)
	}

	// Write recorded attributes
	r.Attrs(func(a slog.Attr) bool {
		h.walkAttr(h.groups, h.prefix, a, func(prefix string, a slog.Attr) {
			h.writeAttrLineStart(buf, r.Level)
			appendKeyValue(buf, prefix, a)
			h.colorEnd(buf)
		})
		return true
	})

	buf.WriteByte('\n')

	// Set output, slog levels are open-ended so anything at or above
	// ERROR belongs to the error writer
	output := h.w.Out
	if r.Level >= slog.LevelError {
		output = h.w.Err
	}

	// Write with lock to avoid race conditions
	h.mu.Lock()
	_, err := output.Write(buf.Bytes())
	h.mu.Unlock() // defer is expensive, not required in this case

	// Return the buffer to the pool if it hasn't grown too large,
	// otherwise let the GC collect it.
	if buf.Cap() <= maxBufferSize {
		bufPool.Put(buf)
	}

	return err
}

// Cancel invokes the cancel function attached to the handler, if any, and
// reports whether one was called. It backs logging.Critical.
func (h *Handler) Cancel() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.cancel == nil {
		return false
	}

	h.cancel()
	return true
}

// walkAttr resolves a, applies the configured ReplaceAttr, and expands
// group values, calling emit for each retained leaf attribute with its
// dot-qualified key prefix. Per the slog.Handler contract, empty
// attributes are elided, groups with an empty key are inlined, and groups
// without attributes are dropped.
func (h *Handler) walkAttr(
	gs []string, prefix string, a slog.Attr,
	emit func(prefix string, a slog.Attr),
) {
	a.Value = a.Value.Resolve()
	if rep := h.opt.ReplaceAttr; rep != nil &&
		a.Value.Kind() != slog.KindGroup {
		a = rep(gs, a)
		// ReplaceAttr may itself return an unresolved value
		a.Value = a.Value.Resolve()
	}

	if a.Equal(slog.Attr{}) {
		return
	}

	if a.Value.Kind() != slog.KindGroup {
		emit(prefix, a)
		return
	}

	attrs := a.Value.Group()
	if len(attrs) == 0 {
		return
	}
	if a.Key != "" {
		// The full slice expression forces a copy: appending in place
		// could write into the backing array shared with other
		// handlers walking their groups concurrently.
		gs = append(gs[:len(gs):len(gs)], a.Key)
		prefix += a.Key + "."
	}
	for _, ga := range attrs {
		h.walkAttr(gs, prefix, ga, emit)
	}
}

// colorize writes s to b wrapped in the given ANSI color code.
// If colored output is disabled, the string is written as-is.
func (h *Handler) colorize(b *bytes.Buffer, c color, s string) {
	h.colorStart(b, c)
	b.WriteString(s)
	h.colorEnd(b)
}

// colorStart begins an ANSI colored span when colors are enabled.
func (h *Handler) colorStart(b *bytes.Buffer, c color) {
	if h.opt.Colored {
		b.WriteString(string(c))
	}
}

// colorEnd closes an ANSI colored span when colors are enabled.
func (h *Handler) colorEnd(b *bytes.Buffer) {
	if h.opt.Colored {
		b.WriteString(string(colorReset))
	}
}

// writeSyslogPrefix prepends a syslog severity prefix to the buffer
// when ForceSyslog is enabled.
func (h *Handler) writeSyslogPrefix(b *bytes.Buffer, l slog.Level) {
	if h.opt.ForceSyslog {
		b.WriteString(syslogPrefix(l))
	}
}

// writeSource appends the "file:line: " reference in bright grey.
func (h *Handler) writeSource(b *bytes.Buffer, pc uintptr) {
	s := source(h.opt.BasePath, pc)
	if s.File == "" {
		s.File = "???"
	}

	h.colorStart(b, colorBrightGrey)
	b.WriteString(s.File)
	b.WriteByte(':')
	b.Write(strconv.AppendInt(b.AvailableBuffer(), int64(s.Line), 10))
	b.WriteString(": ")
	h.colorEnd(b)
}

// writeAttrLineStart begins an attribute line: newline, optional syslog
// prefix, and the indent marker opening a bright grey span the caller
// must close with colorEnd.
func (h *Handler) writeAttrLineStart(b *bytes.Buffer, l slog.Level) {
	b.WriteByte('\n')
	h.writeSyslogPrefix(b, l)
	h.colorStart(b, colorBrightGrey)
	b.WriteString("\t- ")
}

// appendKeyValue writes "key=value" with the group qualification prefix.
func appendKeyValue(b *bytes.Buffer, prefix string, a slog.Attr) {
	appendEscaped(b, prefix)
	appendEscaped(b, a.Key)
	b.WriteByte('=')
	appendValue(b, a.Value)
}

// appendValue writes a resolved value using strconv appenders to avoid
// the allocations of Value.String on the hot path.
func appendValue(b *bytes.Buffer, v slog.Value) {
	switch v.Kind() {
	case slog.KindString:
		appendEscaped(b, v.String())
	case slog.KindInt64:
		b.Write(strconv.AppendInt(b.AvailableBuffer(), v.Int64(), 10))
	case slog.KindUint64:
		b.Write(strconv.AppendUint(b.AvailableBuffer(), v.Uint64(), 10))
	case slog.KindFloat64:
		b.Write(strconv.AppendFloat(
			b.AvailableBuffer(), v.Float64(), 'g', -1, 64))
	case slog.KindBool:
		b.Write(strconv.AppendBool(b.AvailableBuffer(), v.Bool()))
	case slog.KindDuration:
		b.WriteString(v.Duration().String())
	case slog.KindTime:
		b.Write(v.Time().AppendFormat(
			b.AvailableBuffer(), time.RFC3339Nano))
	default:
		appendEscaped(b, v.String())
	}
}

// escapeGroupName quotes a group name containing control bytes, with
// the same strconv semantics as appendEscaped. It runs when a group is
// derived, not per record.
func escapeGroupName(s string) string {
	if !needsEscape(s) {
		return s
	}

	return strconv.Quote(s)
}

// appendEscaped writes s, quoting it with strconv semantics when it
// contains control bytes, so hostile values can neither forge log lines
// (CWE-117) nor inject terminal escape sequences (CWE-150).
func appendEscaped(b *bytes.Buffer, s string) {
	if !needsEscape(s) {
		b.WriteString(s)
		return
	}

	b.Write(strconv.AppendQuote(b.AvailableBuffer(), s))
}

// needsEscape reports whether s contains a control byte. Those byte
// values never occur inside multi-byte UTF-8 sequences, so a byte scan
// is safe.
func needsEscape(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] == 0x7f {
			return true
		}
	}

	return false
}
