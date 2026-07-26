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

// gormSecretModel is the schema used by the real-database round-trip tests.
type gormSecretModel struct {
	ID     uint
	Secret String
}

// TestGormRoundTrip verifies the full gorm integration against a real sqlite
// database: schema parsing, Create persisting the REAL secret (not the
// obfuscated placeholder), and First reading it back into a String.
func TestGormRoundTrip(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	if err := db.AutoMigrate(&gormSecretModel{}); err != nil {
		t.Fatalf("AutoMigrate() error: %v", err)
	}

	const secret = "S3cret-roundtrip"

	in := gormSecretModel{Secret: NewString(secret)}
	if err := db.Create(&in).Error; err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	// The real secret must be stored in the database. The dangerous
	// serializer:json workaround silently persisted "***" instead.
	var raw string
	err = db.Raw(
		"SELECT secret FROM gorm_secret_models WHERE id = ?", in.ID,
	).Scan(&raw).Error
	if err != nil {
		t.Fatalf("raw SELECT error: %v", err)
	}
	if raw != secret {
		t.Errorf("stored raw value = %q, want %q", raw, secret)
	}

	var out gormSecretModel
	if err := db.First(&out, in.ID).Error; err != nil {
		t.Fatalf("First() error: %v", err)
	}

	if got := out.Secret.Reveal(); got != secret {
		t.Errorf("read-back Reveal() = %q, want %q", got, secret)
	}

	// The value read back from the database must still obfuscate everywhere.
	if got := fmt.Sprint(out.Secret); got != obfuscated {
		t.Errorf("read-back fmt.Sprint = %q, want %q", got, obfuscated)
	}
}

// TestGormRoundTripEmpty verifies that an empty secret survives the
// database round-trip.
func TestGormRoundTripEmpty(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm db: %v", err)
	}

	if err := db.AutoMigrate(&gormSecretModel{}); err != nil {
		t.Fatalf("AutoMigrate() error: %v", err)
	}

	in := gormSecretModel{Secret: NewString("")}
	if err := db.Create(&in).Error; err != nil {
		t.Fatalf("Create() error: %v", err)
	}

	var out gormSecretModel
	if err := db.First(&out, in.ID).Error; err != nil {
		t.Fatalf("First() error: %v", err)
	}

	if !out.Secret.IsEmpty() {
		t.Errorf("read-back IsEmpty() = false, want true")
	}
}

// TestStringScan verifies the sql.Scanner contract on *String for every
// source type a driver may hand over, including unsupported ones.
func TestStringScan(t *testing.T) {
	tests := []struct {
		name      string
		give      any
		want      string
		shouldErr bool
	}{
		{name: "string", give: "from-string", want: "from-string"},
		{name: "bytes", give: []byte("from-bytes"), want: "from-bytes"},
		{name: "nil", give: nil, want: ""},
		{name: "empty-string", give: "", want: ""},
		{name: "int64-unsupported", give: int64(42), shouldErr: true},
		{name: "bool-unsupported", give: true, shouldErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s String
			err := s.Scan(tt.give)

			if tt.shouldErr {
				if err == nil {
					t.Fatalf("Scan(%v) error = nil, want error", tt.give)
				}
				return
			}

			if err != nil {
				t.Fatalf("Scan(%v) error: %v", tt.give, err)
			}
			if got := s.Reveal(); got != tt.want {
				t.Errorf("after Scan(%v): Reveal() = %q, want %q", tt.give, got, tt.want)
			}
		})
	}
}

// TestStringScanCopiesBytes verifies that Scan copies driver-owned []byte
// memory, as required by the sql.Scanner documentation.
func TestStringScanCopiesBytes(t *testing.T) {
	src := []byte("driver-owned")

	var s String
	if err := s.Scan(src); err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	// The driver reuses its buffer after Scan returns.
	copy(src, "clobbered!!!")

	if got := s.Reveal(); got != "driver-owned" {
		t.Errorf("after driver buffer reuse: Reveal() = %q, want %q", got, "driver-owned")
	}
}
