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
	"errors"
	"log/slog"
	"os"
	"runtime/debug"

	"gitlab.com/iglou.eu/goulc/logging/model"
)

const (
	ErrWriterOutNil   = "out writer is nil, this is probably a mistake"
	ErrLogLevelUnknow = "Unknow log level provided"
)

// DefaultWriter provides the standard output configuration where
// normal logs go to os.Stdout and error logs to os.Stderr
var DefaultWriter = &model.Writer{Out: os.Stdout, Err: os.Stderr}

// DefaultConfig provides the default logging configuration:
// - Level: "INFO" (only INFO and above are logged)
// - Colored: false (no ANSI colors in output)
// - AddSource: true (includes source file and line information)
var DefaultConfig = &model.Config{Level: "INFO", Colored: false, AddSource: true}

// New is a constructor for the Logger type.
// Same as NewWithWriter with the default writer.
func New(basePath string, cfg *model.Config) (*slog.Logger, error) {
	return NewWithWriter(basePath, DefaultWriter, cfg)
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
func NewWithWriter(basePath string, writer *model.Writer, cfg *model.Config) (*slog.Logger, error) {
	if writer == nil {
		writer = DefaultWriter
	}

	if writer.Out == nil {
		return nil, errors.New(ErrWriterOutNil)
	}

	if writer.Err == nil {
		writer.Err = writer.Out
	}

	if cfg == nil {
		cfg = DefaultConfig
	}

	level, err := getLevel(cfg.Level)
	if err != nil {
		return nil, err
	}

	return slog.New(NewHandler(
		cfg.Cancel,
		writer,
		&HandlerOptions{
			Colored:   cfg.Colored,
			AddSource: cfg.AddSource,
			BasePath:  basePath,
		},
		&slog.HandlerOptions{
			AddSource: cfg.AddSource,
			Level:     level,
		},
	)), nil
}

// Critical logs a critical error message along with any provided attributes,
// prints the stack trace, and then terminates the program with exit code 1.
// The function accepts a Logger instance, a message string, and optional attributes.
func Critical(l *slog.Logger, msg string, attrs ...any) {
	l.With(attrs...).Error(
		"Critical error",
		"error message", msg,
		"stacktrace", string(debug.Stack()),
	)

	logging, ok := l.Handler().(*Handler)
	if !ok || !logging.Cancel() {
		os.Exit(1)
	}
}

func getLevel(level string) (slog.Level, error) {
	switch level {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, errors.New(ErrLogLevelUnknow + ": " + level)
	}
}
