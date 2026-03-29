//go:build windows

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
)

// Priority is a no-op stub for compilation compatibility on Windows,
// where log/syslog is unavailable.
type Priority int

// SyslogPrefix is a no-op stub, always returns an empty string on Windows.
func (p Priority) SyslogPrefix() string {
	return ""
}

// IsSyslog is a no-op stub, always returns false on Windows.
func IsSyslog() bool {
	return false
}

// SyslogSeverity is a no-op stub, always returns 0 on Windows.
func SyslogSeverity(l slog.Level) Priority {
	return 0
}

// BuildSyslogPrefix is a no-op stub, always returns an empty string
// on Windows.
func BuildSyslogPrefix(l slog.Level) string {
	return ""
}
