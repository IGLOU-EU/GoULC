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
	"log/slog"
	"runtime"
	"strconv"
	"strings"
)

// color represents an ANSI escape code used for terminal color output.
type color string

const (
	colorReset        color = "\033[0m"
	colorRed          color = "\033[31m"
	colorCyan         color = "\033[36m"
	colorMagenta      color = "\033[35m"
	colorBrightGrey   color = "\033[90m"
	colorBrightYellow color = "\033[93m"
)

// setColorLevel returns the ANSI color associated with the given log level.
// Unknown levels default to magenta.
func setColorLevel(l slog.Level) color {
	switch l {
	case slog.LevelDebug:
		return colorBrightGrey
	case slog.LevelInfo:
		return colorCyan
	case slog.LevelWarn:
		return colorBrightYellow
	case slog.LevelError:
		return colorRed
	}

	return colorMagenta
}

// sourceBuilder formats a source code location as "file:line: " for use
// in log output prefixes.
func sourceBuilder(file string, line int) string {
	var source strings.Builder
	source.WriteString(file)
	source.WriteRune(':')
	source.WriteString(strconv.Itoa(line))
	source.WriteString(": ")
	return source.String()
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
