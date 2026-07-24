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

// Package contract declares idiomatic named interfaces for common method
// conventions that the standard library leaves implicit or unexported.
//
// Declaring them once allows two things across the rest of GoULC:
//   - explicit compile-time conformance assertions
//     (var _ contract.IsZeroer = T{}) that break the build if a signature
//     drifts, instead of unreadable anonymous interfaces;
//   - reflect-free generic checks through a type assertion
//     (if z, ok := v.(contract.IsZeroer); ok { ... }) instead of the costlier
//     reflect.Value.IsZero.
//
// This package is a leaf: it imports nothing, so any other package can depend
// on it without risking an import cycle. Add a new interface only when a real
// need arises, never speculatively.
package contract

// IsZeroer is implemented by types that report whether they hold their zero
// value. It mirrors the unexported encoding/json.isZeroer interface used by the
// ",omitzero" struct tag (Go 1.24+) and the time.Time.IsZero convention.
type IsZeroer interface {
	IsZero() bool
}

// Emptier is implemented by types that report semantic emptiness, which may
// differ from the zero value (for example a non-nil but length-zero
// container). The standard library defines no equivalent interface.
type Emptier interface {
	IsEmpty() bool
}
