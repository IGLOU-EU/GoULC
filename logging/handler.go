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
	"sync"

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
	h   slog.Handler
	w   model.Writer
	opt HandlerOptions

	group             string
	preformattedAttrs []string

	cancel context.CancelFunc
	mu     *sync.RWMutex
}

var _ = slog.Handler(&Handler{})

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
// If opt or sopt is nil, default options will be used.
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
		h:   slog.NewTextHandler(w.Out, &localOpt.HandlerOptions),
		w:   *w,
		opt: localOpt,

		cancel: cancel,
		mu:     &sync.RWMutex{},
	}
}

// Enabled implements slog.Handler interface and determines if a log level
// should be processed based on the handler's configuration
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

// WithAttrs implements slog.Handler interface and returns a new Handler with
// the given attributes added to the set of attributes that will be logged
// with each log record. The attributes are stored as a slice of strings.
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Lock during copy
	h.mu.RLock()

	// Copy and initialize new handler
	oldAttrsLen := len(h.preformattedAttrs)
	newHandler := &Handler{
		h:   h.h,
		w:   h.w,
		opt: h.opt,

		group:             h.group,
		preformattedAttrs: make([]string, oldAttrsLen, oldAttrsLen+len(attrs)),

		cancel: h.cancel,
		mu:     &sync.RWMutex{},
	}

	// Copy existing preformatted attributes
	copy(newHandler.preformattedAttrs, h.preformattedAttrs)

	// The copy is done, unlock
	h.mu.RUnlock()

	// Add new attributes
	for _, attr := range attrs {
		if !attr.Equal(slog.Attr{}) {
			newHandler.preformattedAttrs = append(
				newHandler.preformattedAttrs, attr.String())
		}
	}

	return newHandler
}

// WithGroup returns a new handler with the given group name appended.
// This implementation uses a simple dot-separated prefix rather than
// nested group handling, since slog.Group covers most grouping needs.
func (h *Handler) WithGroup(group string) slog.Handler {
	h.mu.RLock()
	newHandler := &Handler{
		h:   h.h,
		w:   h.w,
		opt: h.opt,

		group:             h.group + "." + group,
		preformattedAttrs: append([]string(nil), h.preformattedAttrs...),

		cancel: h.cancel,
		mu:     &sync.RWMutex{},
	}
	h.mu.RUnlock()

	return newHandler
}

// Handle implements slog.Handler interface. It formats and writes a log record.
// The implementation:
// - Applies ANSI colors if enabled
// - Uses different writers for error and non-error logs
// - Includes timestamp, level, source location (if enabled), and message
// - Formats and writes all record attributes
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

	// Date and time
	buf.WriteString(r.Time.Format(h.opt.TimeFormat))
	buf.WriteByte(' ')

	// Level
	buf.WriteByte('[')
	h.colorize(buf, colorLevel, r.Level.String())
	buf.WriteByte(']')
	buf.WriteByte(' ')

	// Prefix
	if h.group != "" {
		buf.WriteByte('[')
		buf.WriteString("G:")
		h.colorize(buf, colorLevel, h.group)
		buf.WriteByte(']')
		buf.WriteByte(' ')
	}

	// Source
	if h.opt.Config.AddSource {
		s := source(h.opt.BasePath, r.PC)
		if s.File == "" {
			s.File = "???"
		}

		h.colorize(buf, colorBrightGrey, sourceBuilder(s.File, s.Line))
	}

	// Message
	buf.WriteString(r.Message)

	// Write preformatted attributes
	for _, a := range h.preformattedAttrs {
		h.writeAttributes(buf, r.Level, a)
	}

	// Write recorded attributes
	r.Attrs(func(a slog.Attr) bool {
		return h.writeAttributes(buf, r.Level, a.String())
	})

	buf.WriteByte('\n')

	// Set output
	output := h.w.Out
	if r.Level == slog.LevelError {
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

// cancel is a function of logging.Handler for working with logging.Critical
// it permits to cancel the context and terminate the program
func (h *Handler) Cancel() bool {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.cancel == nil {
		return false
	}

	h.cancel()
	return true
}

// colorize writes s to b wrapped in the given ANSI color code.
// If colored output is disabled, the string is written as-is.
func (h *Handler) colorize(b *bytes.Buffer, c color, s string) {
	if !h.opt.Colored {
		b.WriteString(s)
		return
	}

	b.WriteString(string(c))
	b.WriteString(s)
	b.WriteString(string(colorReset))
}

// writeSyslogPrefix prepends a syslog severity prefix to the buffer
// when ForceSyslog is enabled.
func (h *Handler) writeSyslogPrefix(b *bytes.Buffer, l slog.Level) {
	if h.opt.ForceSyslog {
		b.WriteString(BuildSyslogPrefix(l))
	}
}

// writeAttributes writes a single attribute line to the buffer, prefixed with
// a syslog header (if enabled) and styled in bright grey. It always returns
// true to satisfy the slog.Record.Attrs callback signature.
func (h *Handler) writeAttributes(
	b *bytes.Buffer, l slog.Level, s string,
) bool {
	b.WriteByte('\n')
	h.writeSyslogPrefix(b, l)
	h.colorize(b, colorBrightGrey, "\t- "+s)

	return true
}
