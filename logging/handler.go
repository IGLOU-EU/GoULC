/*
 * Copyright 2024 Adrien Kara
 *
 * This program is free software: you can redistribute it and/or modify
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
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package logging

import (
	"bytes"
	"context"
	"log/slog"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"gitlab.com/iglou.eu/goulc/logging/model"
)

const (
	colorReset        = "\033[0m"
	colorRed          = "\033[31m"
	colorCyan         = "\033[36m"
	colorMagenta      = "\033[35m"
	colorBrightGrey   = "\033[90m"
	colorBrightYellow = "\033[93m"
)

var TimeFormat = "[2006-01-02 15:04:05]"

// Handler implements slog.Handler interface with additional features:
// - Colored output support with ANSI colors
// - Separate writers for normal and error logs
// - Concurrent-safe logging with mutex protection
// - Source code location with customizable base path
type Handler struct {
	h    slog.Handler
	w    model.Writer
	opts HandlerOptions

	group             string
	preformattedAttrs []string

	cancel context.CancelFunc
	mu     *sync.RWMutex
}

// HandlerOptions configures the behavior of the Handler.
type HandlerOptions struct {
	// Colored enables ANSI color output in log messages
	Colored bool
	// AddSource includes the source file and line number in log messages
	AddSource bool
	// BasePath is used to trim the full file path in source code references
	// For example, if BasePath is "/home/user/project", a source file
	// "/home/user/project/pkg/file.go" will be shown as "pkg/file.go"
	BasePath string
}

// Enabled implements slog.Handler interface and determines if a log level
// should be processed based on the handler's configuration
func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.h.Enabled(ctx, level)
}

// WithAttrs implements slog.Handler interface and returns a new Handler with
// the given attributes added to the set of attributes that will be logged
// with each log record. The attributes are stored as a slice of strings.
//
// Attrs are stored in pr
func (h *Handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// Lock during copy
	h.mu.RLock()

	// Copy and initialize new handler
	oldAttrsLen := len(h.preformattedAttrs)
	new := &Handler{
		h:    h.h,
		w:    h.w,
		opts: h.opts,

		group:             h.group,
		preformattedAttrs: make([]string, oldAttrsLen, oldAttrsLen+len(attrs)),

		cancel: h.cancel,
		mu:     &sync.RWMutex{},
	}

	// Copy existing preformatted attributes
	copy(new.preformattedAttrs, h.preformattedAttrs)

	// The copy is done, unlock
	h.mu.RUnlock()

	// Add new attributes
	for _, attr := range attrs {
		if !attr.Equal(slog.Attr{}) {
			new.preformattedAttrs = append(new.preformattedAttrs, attr.String())
		}
	}

	return new
}

// WithGroup returns a new Logger with the given group added to the group stack.
// groups are dot-separated: if the handler has group "a" and WithGroup("b") is
// called, the new handler will have group ".a.b".
//
// I don't like the regular implementation, there is require more complexity
// than I need. For grouping, I prefer the usage of anonymous structs, maps
// or better yet, slog.Group. And it can be combined with WithAttrs.
func (h *Handler) WithGroup(group string) slog.Handler {
	h.mu.RLock()
	new := &Handler{
		h:    h,
		w:    h.w,
		opts: h.opts,

		group:             h.group + "." + group,
		preformattedAttrs: h.preformattedAttrs,

		cancel: h.cancel,
		mu:     &sync.RWMutex{},
	}
	h.mu.RUnlock()

	return new
}

// Handle implements slog.Handler interface. It formats and writes a log record.
// The implementation:
// - Applies ANSI colors if enabled
// - Uses different writers for error and non-error logs
// - Includes timestamp, level, source location (if enabled), and message
// - Formats and writes all record attributes
// - Is concurrent-safe through mutex protection
func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	// Set colors if enabled
	var cLevel, cReset, cAttrs string
	if h.opts.Colored {
		cReset = colorReset
		cAttrs = colorBrightGrey

		switch r.Level {
		case slog.LevelDebug:
			cLevel = colorBrightGrey
		case slog.LevelInfo:
			cLevel = colorCyan
		case slog.LevelWarn:
			cLevel = colorBrightYellow
		case slog.LevelError:
			cLevel = colorRed
		default:
			cLevel = colorMagenta
		}
	}

	// Init buffer
	buf := bytes.NewBuffer(nil)
	buf.Grow(1024) // Allocate a default size

	// Date and time
	buf.WriteString(time.Now().Format(TimeFormat))
	buf.WriteRune(' ')

	// Level
	buf.WriteRune('[')
	buf.WriteString(cLevel)
	buf.WriteString(r.Level.String())
	buf.WriteString(cReset)
	buf.WriteRune(']')
	buf.WriteRune(' ')

	// Prefix
	if h.group != "" {
		buf.WriteRune('[')
		buf.WriteString("G:")
		buf.WriteString(cLevel)
		buf.WriteString(h.group)
		buf.WriteString(cReset)
		buf.WriteRune(']')
		buf.WriteRune(' ')
	}

	// Source
	if h.opts.AddSource {
		s := source(h.opts.BasePath, r.PC)
		if s.File == "" {
			s.File = "???"
		}

		buf.WriteString(cAttrs)
		buf.WriteString(s.File)
		buf.WriteRune(':')
		buf.WriteString(strconv.Itoa(s.Line))
		buf.WriteRune(':')
		buf.WriteString(cReset)
		buf.WriteRune(' ')
	}

	// Message
	buf.WriteString(r.Message)

	// Attributes
	buf.WriteString(cAttrs)

	// Write preformatted attributes
	for _, a := range h.preformattedAttrs {
		buf.WriteRune('\n')
		buf.WriteString("	- " + a)
	}

	// Write recorded attributes
	r.Attrs(func(a slog.Attr) bool {
		buf.WriteRune('\n')
		buf.WriteString(
			"	- " + a.String(),
		)

		return true
	})

	buf.WriteString(cReset)
	buf.WriteString("\n")

	// Set output
	output := h.w.Out
	if r.Level == slog.LevelError {
		output = h.w.Err
	}

	// Write with lock to avoid race conditions
	h.mu.Lock()
	_, err := output.Write(buf.Bytes())
	h.mu.Unlock() // defer is expensive, not required in this case

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

// NewHandler creates a new Handler with the given writer and options.
// The writer specifies where to write logs (separate streams for normal and error logs).
// The opts parameter configures coloring, source code info, and base path.
// The sopts parameter configures slog-specific handler options.
//
// If opts or sopts is nil, default options will be used.
// The handler is concurrent-safe and implements the slog.Handler interface.
func NewHandler(cancel context.CancelFunc, w *model.Writer, opts *HandlerOptions, sopts *slog.HandlerOptions) *Handler {
	if opts == nil {
		opts = &HandlerOptions{}
	}

	if sopts == nil {
		sopts = &slog.HandlerOptions{}
	}

	if opts.AddSource {
		if pos := strings.Index(opts.BasePath, "cmd"); pos > 0 {
			opts.BasePath = opts.BasePath[:pos]
		}
	}

	return &Handler{
		h:    slog.NewTextHandler(w.Out, sopts),
		w:    *w,
		opts: *opts,

		cancel: cancel,
		mu:     &sync.RWMutex{},
	}
}

// source returns a Source describing the caller's source code position.
// It trims the file path using basePath to make it more readable.
func source(basePath string, pc uintptr) *slog.Source {
	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	return &slog.Source{
		Function: f.Function,
		File:     strings.TrimPrefix(f.File, basePath),
		Line:     f.Line,
	}
}
