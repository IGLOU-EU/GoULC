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
	"errors"
	"log/slog"
	"os"
	"runtime/debug"
	"strings"

	"gitlab.com/iglou.eu/goulc/logging/model"
)

var (
	// ErrWriterOutNil reports a Writer whose Out stream is nil.
	ErrWriterOutNil = errors.New(
		"out writer is nil, this is probably a mistake")
	// ErrLogLevelUnknown reports a Config.Level outside the known set.
	ErrLogLevelUnknown = errors.New("unknown log level provided")
)

// Config is an alias of the model package type so common usage only
// needs the logging import.
type Config = model.Config

// Writer is an alias of the model package type so common usage only
// needs the logging import.
type Writer = model.Writer

// DefaultWriter provides the standard output configuration where
// normal logs go to os.Stdout and error logs to os.Stderr
func DefaultWriter() *model.Writer {
	return &model.Writer{Out: os.Stdout, Err: os.Stderr}
}

// DefaultConfig provides the default logging configuration:
// - Level: "INFO" (only INFO and above are logged)
// - Colored: false (no ANSI colors in output)
// - AddSource: true (includes source file and line information)
func DefaultConfig() *model.Config {
	return &model.Config{
		Level: "INFO", Colored: false, AddSource: true}
}

// New is a constructor for the Logger type.
// Same as NewWithWriter with the default writer.
func New(basePath string, cfg *model.Config) (*slog.Logger, error) {
	return NewWithWriter(basePath, nil, cfg)
}

// NewWithWriter creates a new logger with a custom writer
//
// The basePath is used for source code reference, to get the file and line
// number of the caller without a full path output.
//
// If writer is nil, the default writer will be used. It return an error in the
// case of writer.Out is nil and use writer.Out as writer.Err if it is nil.
//
// The cfg use the default configuration if nil
func NewWithWriter(
	basePath string, writer *model.Writer, cfg *model.Config,
) (*slog.Logger, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	localCfg := *cfg

	level, err := getLevel(localCfg.Level)
	if err != nil {
		return nil, err
	}

	if writer == nil {
		// When using the default writer we can safely force syslog
		// prefixes if the process is running under systemd's journal.
		// We only do this here because forcing syslog on a user-provided
		// writer could interfere with their output format.
		if IsSyslog() {
			localCfg.ForceSyslog = true
		}

		writer = DefaultWriter()
	}
	w := *writer

	if w.Out == nil {
		return nil, ErrWriterOutNil
	}

	if w.Err == nil {
		w.Err = w.Out
	}

	return slog.New(NewHandler(
		localCfg.Cancel,
		&w,
		&HandlerOptions{
			// The whole Config is carried over so a new field cannot be
			// silently lost in a field-by-field copy.
			Config: localCfg,
			HandlerOptions: slog.HandlerOptions{
				AddSource: localCfg.AddSource,
				Level:     level,
			},

			BasePath: basePath,
		},
	)), nil
}

// Critical logs a critical error message along with any provided
// attributes and the current stack trace, then terminates the program.
//
// Termination is abrupt by design. When the logger's handler is a
// *Handler carrying a cancel function (Config.Cancel), that function is
// called instead and the caller owns the shutdown. In every other case
// Critical calls os.Exit(1), which skips deferred functions and flushes
// nothing. Reserve it for main-level failure handling, never call it
// from library code.
func Critical(l *slog.Logger, msg string, attrs ...any) {
	l.With(attrs...).Error(
		"Critical error",
		"error message", msg,
		"stacktrace", string(debug.Stack()),
	)

	logging, ok := l.Handler().(*Handler)
	if !ok || !logging.Cancel() {
		//nolint:revive
		// This is an urgency exit and can be ignored by the linter
		os.Exit(1)
	}
}

// getLevel converts a log level string ("DEBUG", "INFO", "WARN", "ERROR")
// to its corresponding slog.Level.
// Returns an error if the level string is not recognized.
func getLevel(level string) (slog.Level, error) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, ErrLogLevelUnknown
	}
}
