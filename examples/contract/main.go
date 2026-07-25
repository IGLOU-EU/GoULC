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

package main

import (
	"fmt"

	"gitlab.com/iglou.eu/goulc/contract"
)

// Inventory shows why the two interfaces differ: a non-nil but
// length-zero item list is empty without being the zero value.
type Inventory struct {
	Items []string
}

// Compile-time conformance assertions: the build breaks here if a
// method signature drifts away from the contract.
var (
	_ contract.IsZeroer = Inventory{}
	_ contract.Emptier  = Inventory{}
)

// IsZero reports whether the inventory is the zero value, a nil slice
// here, unlike an allocated but empty one.
func (i Inventory) IsZero() bool {
	return i.Items == nil
}

// IsEmpty reports semantic emptiness, no items to carry, whether the
// slice is nil or allocated.
func (i Inventory) IsEmpty() bool {
	return len(i.Items) == 0
}

func main() {
	describe("zero value", Inventory{})
	describe("empty but not zero", Inventory{Items: []string{}})
	describe("filled", Inventory{Items: []string{"sword", "shield"}})
	describe("plain int", 42)
}

// describe checks the contract interfaces with plain type assertions,
// no reflect involved.
func describe(label string, v any) {
	z, zok := v.(contract.IsZeroer)
	e, eok := v.(contract.Emptier)

	switch {
	case zok && eok:
		fmt.Printf("%-20s IsZero=%-5v IsEmpty=%v\n",
			label+":", z.IsZero(), e.IsEmpty())
	case zok:
		fmt.Printf("%-20s IsZero=%v\n", label+":", z.IsZero())
	case eok:
		fmt.Printf("%-20s IsEmpty=%v\n", label+":", e.IsEmpty())
	default:
		fmt.Printf("%-20s implements no contract interface\n", label+":")
	}
}
