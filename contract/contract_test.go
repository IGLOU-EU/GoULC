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

package contract_test

import (
	"testing"
	"time"

	"gitlab.com/iglou.eu/goulc/contract"
)

// counter is a sample type satisfying both contracts.
type counter struct {
	n int
}

func (c counter) IsZero() bool  { return c.n == 0 }
func (c counter) IsEmpty() bool { return c.n == 0 }

// Compile-time conformance. Asserting time.Time pins IsZeroer to the same
// IsZero() bool convention the standard library uses.
var (
	_ contract.IsZeroer = counter{}
	_ contract.IsZeroer = time.Time{}
	_ contract.Emptier  = counter{}
)

func TestIsZeroer(t *testing.T) {
	tests := []struct {
		name string
		give contract.IsZeroer
		want bool
	}{
		{"zero counter", counter{}, true},
		{"non-zero counter", counter{n: 1}, false},
		{"zero time", time.Time{}, true},
		{"non-zero time", time.Unix(1, 0), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.give.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmptier(t *testing.T) {
	tests := []struct {
		name string
		give contract.Emptier
		want bool
	}{
		{"empty", counter{}, true},
		{"non-empty", counter{n: 1}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.give.IsEmpty(); got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTimeNow(t *testing.T) {
	// time.Now must satisfy the type, so it models the standard clock.
	var _ contract.TimeNow = time.Now

	fixed := time.Unix(42, 0)
	var clock contract.TimeNow = func() time.Time { return fixed }

	if got := clock(); !got.Equal(fixed) {
		t.Errorf("clock() = %v, want %v", got, fixed)
	}
}
