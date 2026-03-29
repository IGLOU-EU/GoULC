//go:build gorm

/*
 * Copyright 2025 Adrien Kara
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
	"fmt"
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// TestGormStringStringer verifies that GormString.String() returns the clear
// value (by design, for gorm SQL parameter passing).
func TestGormStringStringer(t *testing.T) {
	gs := GormString("clear-value")
	if got := gs.String(); got != "clear-value" {
		t.Errorf("GormString.String() = %q, want %q", got, "clear-value")
	}
}

// TestGormStringHiding verifies that GormString.Hiding() returns the
// obfuscated value for log filtering.
func TestGormStringHiding(t *testing.T) {
	gs := GormString("secret")
	if got := gs.Hiding(); got != "***" {
		t.Errorf("GormString.Hiding() = %q, want %q", got, "***")
	}
}

// TestGormStringImplementsGormHider verifies the interface compliance.
func TestGormStringImplementsGormHider(t *testing.T) {
	var _ GormHider = GormString("")
}

// TestGormValueExpr verifies that String.GormValue returns a clause.Expr
// with the correct SQL and the value wrapped as GormString.
func TestGormValueExpr(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	s := NewString("db-secret")
	expr := s.GormValue(context.Background(), db)

	if expr.SQL != "?" {
		t.Errorf("GormValue SQL = %q, want %q", expr.SQL, "?")
	}

	if len(expr.Vars) != 1 {
		t.Fatalf("GormValue Vars len = %d, want 1", len(expr.Vars))
	}

	gs, ok := expr.Vars[0].(GormString)
	if !ok {
		t.Fatalf("GormValue Vars[0] type = %T, want GormString", expr.Vars[0])
	}

	if gs.String() != "db-secret" {
		t.Errorf("GormValue Vars[0] value = %q, want %q", gs.String(), "db-secret")
	}

	if gs.Hiding() != "***" {
		t.Errorf("GormValue Vars[0].Hiding() = %q, want %q", gs.Hiding(), "***")
	}
}

// TestGormStringDoesNotLeakViaFmt verifies that GormString used via fmt verbs
// exposes the clear value (this is expected behavior - GormString is internal).
// The security boundary is at the logger level via GormHider.Hiding().
func TestGormStringFmtBehavior(t *testing.T) {
	gs := GormString("test-value")

	// GormString.String() returns clear value, so fmt uses it
	got := fmt.Sprint(gs)
	if got != "test-value" {
		t.Errorf("fmt.Sprint(GormString) = %q, want %q", got, "test-value")
	}
}

// TestGormValueFromEmptyString verifies GormValue works with empty strings.
func TestGormValueFromEmptyString(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	s := NewString("")
	expr := s.GormValue(context.Background(), db)

	gs := expr.Vars[0].(GormString)
	if gs.String() != "" {
		t.Errorf("GormValue empty: String() = %q, want %q", gs.String(), "")
	}
}
