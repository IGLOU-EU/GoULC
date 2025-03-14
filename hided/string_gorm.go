//go:build gorm

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

package hided

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type GormString string

// GormValue implements gorm.Valuer to safely pass string data to gorm, you
// need to implement gorm.ParamsFilter to keep value secret into your logger
func (s String) GormValue(_ context.Context, _ *gorm.DB) clause.Expr {
	return clause.Expr{
		SQL:  "?",
		Vars: []any{GormString(s)},
	}
}

// String is the Stringer, it returns a clear string representation
func (g GormString) String() string {
	return string(g)
}

// Hiding is to return an obfuscated string
func (_ GormString) Hiding() string {
	return "***"
}
