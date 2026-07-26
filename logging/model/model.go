/* model.go
 *
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

// Package model holds the serializable configuration and writer types
// shared by the logging package and its integrations.
package model

import (
	"context"
	"io"
)

// Config defines the configuration options for the logger.
// It controls the logging level, output coloring, and source code information.
// Fields at their zero value are omitted from the JSON output, so a
// persisted configuration only contains what was actually set.
type Config struct {
	// Colored enables ANSI color output in log messages
	Colored bool `json:"colored,omitzero"`
	// AddSource includes the source file and line number in log messages
	AddSource bool `json:"add_source,omitzero"`
	// ForceSyslog prepends syslog severity prefixes (e.g. <6>) to each
	// log line and disables colored output.
	ForceSyslog bool `json:"force_syslog,omitzero"`

	// Level defines the minimum logging level
	// ("DEBUG", "INFO", "WARN", "ERROR")
	Level string `json:"level,omitzero"`
	// TimeFormat defines the layout string passed to time.Format for
	// formatting timestamps in log messages. It must follow Go's reference
	// time layout convention (Mon Jan 2 15:04:05 MST 2006).
	TimeFormat string `json:"time_format,omitzero"`

	// Cancel is a context.CancelFunc used to cancel a global context
	// in case of critical errors
	Cancel context.CancelFunc `json:"-"`
}

// Writer defines the output destinations for different log levels.
// It allows separating normal logs from error logs.
type Writer struct {
	// Out is the writer for normal log messages (DEBUG, INFO, WARN levels)
	Out io.Writer
	// Err is the writer for error messages (ERROR level)
	Err io.Writer
}
