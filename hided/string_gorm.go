//go:build gorm

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

package hided

import (
	"context"
	"database/sql"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

// GormString wraps the plaintext secret handed to gorm as a bind parameter,
// so the REAL value reaches the database while a GormHider-aware logger can
// still obfuscate it.
//
// SECURITY WARNING: GormString.String() intentionally returns the clear
// value, and gorm's DEFAULT logger prints bind parameters in its SQL traces.
// Using hided values with the default logger leaks every secret in clear
// text in the logs. Always configure a logger whose gorm.ParamsFilter
// replaces GormHider parameters with Hiding(), such as
// gitlab.com/iglou.eu/goulc/logging.GormLogger.
type GormString string

var (
	_ GormHider                    = GormString("")
	_ gorm.Valuer                  = String{}
	_ sql.Scanner                  = (*String)(nil)
	_ schema.GormDataTypeInterface = String{}
)

// GormDataType implements schema.GormDataTypeInterface so gorm's schema
// parser maps the field to the dialect's string column type instead of
// rejecting the struct as a broken relation.
func (_ String) GormDataType() string {
	return string(schema.String)
}

// GormValue implements gorm.Valuer to pass the REAL secret to the database
// as a bind parameter, wrapped in GormString so a gorm.ParamsFilter logger
// can still obfuscate it. See the GormString warning about gorm's default
// logger leaking bind parameters.
func (s String) GormValue(_ context.Context, _ *gorm.DB) clause.Expr {
	return clause.Expr{
		SQL:  "?",
		Vars: []any{GormString(s.Reveal())},
	}
}

// Scan implements sql.Scanner so gorm can load a stored secret back into a
// String when reading rows.
func (s *String) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		s.val = nil
	case string:
		s.val = []byte(v)
	case []byte:
		// Copy: the driver owns src and may reuse it after Scan returns.
		s.val = make([]byte, len(v))
		copy(s.val, v)
	default:
		return fmt.Errorf("hided: cannot scan %T into String", src)
	}

	return nil
}

// String implements the Stringer interface and returns a clear string
// representation.
func (g GormString) String() string {
	return string(g)
}

// Hiding returns an obfuscated string.
func (_ GormString) Hiding() string {
	return obfuscated
}
