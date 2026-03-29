//go:build !windows

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
	"log/slog"
	"log/syslog"
	"os"
	"strconv"
)

// Priority is a syslog priority level wrapping log/syslog.Priority.
type Priority syslog.Priority

// SyslogPrefix returns the syslog priority
// formatted as a prefix string like "<6>"
func (p Priority) SyslogPrefix() string {
	return "<" + strconv.Itoa(int(p)) + ">"
}

// IsSyslog reports whether the process is running under systemd's journal
// by checking the JOURNAL_STREAM environment variable.
func IsSyslog() bool {
	return os.Getenv("JOURNAL_STREAM") != ""
}

// SyslogSeverity maps a slog.Level to its corresponding syslog Priority
// Unknown levels default to LOG_INFO.
func SyslogSeverity(l slog.Level) Priority {
	switch l {
	case slog.LevelDebug:
		return Priority(syslog.LOG_DEBUG)
	case slog.LevelInfo:
		return Priority(syslog.LOG_INFO)
	case slog.LevelWarn:
		return Priority(syslog.LOG_WARNING)
	case slog.LevelError:
		return Priority(syslog.LOG_ERR)
	}

	return Priority(syslog.LOG_INFO)
}

// BuildSyslogPrefix returns a ready-to-use syslog prefix string
// for the given log level.
func BuildSyslogPrefix(l slog.Level) string {
	return SyslogSeverity(l).SyslogPrefix()
}
