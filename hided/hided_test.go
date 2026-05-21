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

package hided

import (
	"testing"
)

// intHider is a minimal Hider implementation whose underlying value is an int.
// It is used to verify Value[T] against a non-string Hider.
type intHider struct {
	v int
}

func (_ intHider) String() string  { return obfuscated }
func (i intHider) IsEmpty() bool   { return i.v == 0 }
func (_ intHider) HashMD5() string { return "" }
func (i intHider) Value() any      { return i.v }

// TestValueGeneric verifies the generic Value[T] helper across matching,
// mismatching, and edge-case type parameters.
func TestValueGeneric(t *testing.T) {
	t.Run("string-match", func(t *testing.T) {
		s := NewString("the-real-secret")
		got := Value[string](s)
		if got != "the-real-secret" {
			t.Errorf("Value[string] = %q, want %q", got, "the-real-secret")
		}
	})

	t.Run("string-empty", func(t *testing.T) {
		s := NewString("")
		got := Value[string](s)
		if got != "" {
			t.Errorf("Value[string] on empty = %q, want %q", got, "")
		}
	})

	t.Run("type-mismatch-returns-zero", func(t *testing.T) {
		s := NewString("not-an-int")
		got := Value[int](s)
		if got != 0 {
			t.Errorf("Value[int] on String = %d, want 0 (zero value)", got)
		}
	})

	t.Run("type-mismatch-returns-zero-bool", func(t *testing.T) {
		s := NewString("not-a-bool")
		got := Value[bool](s)
		if got {
			t.Errorf("Value[bool] on String = %v, want false (zero value)", got)
		}
	})

	t.Run("type-mismatch-returns-zero-bytes", func(t *testing.T) {
		s := NewString("not-bytes")
		got := Value[[]byte](s)
		if got != nil {
			t.Errorf("Value[[]byte] on String = %v, want nil (zero value)", got)
		}
	})

	t.Run("any-target-type", func(t *testing.T) {
		s := NewString("via-any")
		got := Value[any](s)
		str, ok := got.(string)
		if !ok {
			t.Fatalf("Value[any] returned %T, want string under any", got)
		}
		if str != "via-any" {
			t.Errorf("Value[any] = %q, want %q", str, "via-any")
		}
	})

	t.Run("non-string-hider", func(t *testing.T) {
		h := intHider{v: 42}
		got := Value[int](h)
		if got != 42 {
			t.Errorf("Value[int] on intHider = %d, want 42", got)
		}
	})

	t.Run("non-string-hider-wrong-target", func(t *testing.T) {
		h := intHider{v: 42}
		got := Value[string](h)
		if got != "" {
			t.Errorf("Value[string] on intHider = %q, want \"\" (zero value)", got)
		}
	})
}

// TestHiderInterfaceCompliance verifies that the canonical hided types
// satisfy the Hider interface contract at compile time.
func TestHiderInterfaceCompliance(t *testing.T) {
	var _ Hider = String{}
	var _ Hider = intHider{}
}
