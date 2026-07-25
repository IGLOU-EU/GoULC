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

// Package logging provides a human-oriented slog.Handler with colored
// output, separate writers for normal and error records, syslog severity
// prefixes, and trimmed source code references.
//
// The handler follows the slog.Handler contract, verified against
// testing/slogtest: attribute values are resolved, so slog.LogValuer
// implementations can redact secrets, HandlerOptions.ReplaceAttr is
// applied to every attribute coming from records or WithAttrs, empty
// attributes and empty groups are elided, a zero time and a zero PC are
// not printed, and WithGroup qualifies attribute keys with a
// dot-separated prefix.
//
// Known deviations from the canonical slog handlers, assumed by this
// custom text format:
//   - ReplaceAttr is never called for the built-in time, level, source,
//     and message parts. The format lays them out itself, so they cannot
//     be renamed or removed, only attributes can.
//   - Groups are rendered flat, not nested: group membership shows as a
//     dotted key prefix ("grp.key=value") and the active WithGroup path
//     is echoed as a "[G:grp]" marker on the record line.
//   - Attributes are printed one per line after the message line, every
//     line repeating the syslog prefix when ForceSyslog is enabled.
//
// Attribute keys, textual values, and the record message are escaped on
// emission: any string containing a control byte (below 0x20, or 0x7f)
// is quoted with strconv semantics. Untrusted input can therefore
// neither forge log lines nor inject terminal escape sequences.
package logging
