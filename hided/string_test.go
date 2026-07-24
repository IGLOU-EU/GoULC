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
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"
)

// TestNewString verifies that NewString creates a String correctly for both
// empty and non-empty inputs.
func TestNewString(t *testing.T) {
	t.Run("non-empty", func(t *testing.T) {
		s := NewString("secret")
		if s.Value() != "secret" {
			t.Errorf("NewString(\"secret\").Value() = %q, want %q", s.Value(), "secret")
		}
	})

	t.Run("empty", func(t *testing.T) {
		s := NewString("")
		if s.Value() != "" {
			t.Errorf("NewString(\"\").Value() = %q, want %q", s.Value(), "")
		}
	})

	t.Run("unicode", func(t *testing.T) {
		s := NewString("héllo wörld 🔑")
		if s.Value() != "héllo wörld 🔑" {
			t.Errorf("NewString with unicode: Value() = %q, want %q", s.Value(), "héllo wörld 🔑")
		}
	})
}

// TestStringMethod verifies that String() always returns the obfuscated value.
func TestStringMethod(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"non-empty", "secret-password"},
		{"empty", ""},
		{"spaces", "   "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewString(tt.input)
			if got := s.String(); got != obfuscated {
				t.Errorf("String() = %q, want %q", got, obfuscated)
			}
		})
	}
}

// TestGoString verifies that GoString() returns "***", preventing %#v leaks.
func TestGoString(t *testing.T) {
	s := NewString("super-secret")
	if got := s.GoString(); got != obfuscated {
		t.Errorf("GoString() = %q, want %q", got, obfuscated)
	}
}

// TestFmtFormat verifies that all fmt verbs produce "***" via the Format method.
func TestFmtFormat(t *testing.T) {
	s := NewString("do-not-leak")

	verbs := []struct {
		name   string
		format string
	}{
		{"%v", "%v"},
		{"%s", "%s"},
		{"%+v", "%+v"},
		{"%#v", "%#v"},
		{"%q", "%q"},
		{"%x", "%x"},
		{"%d", "%d"},
	}

	for _, v := range verbs {
		t.Run(v.name, func(t *testing.T) {
			got := fmt.Sprintf(v.format, s)
			if got != obfuscated {
				t.Errorf("fmt.Sprintf(%q, s) = %q, want %q", v.format, got, obfuscated)
			}
		})
	}
}

// TestFmtSprintAndSprintln verifies that Sprint and Sprintln do not leak.
func TestFmtSprintAndSprintln(t *testing.T) {
	s := NewString("leak-me-not")

	t.Run("Sprint", func(t *testing.T) {
		got := fmt.Sprint(s)
		if got != obfuscated {
			t.Errorf("fmt.Sprint(s) = %q, want %q", got, obfuscated)
		}
	})

	t.Run("Sprintln", func(t *testing.T) {
		got := fmt.Sprintln(s)
		expected := obfuscated + "\n"
		if got != expected {
			t.Errorf("fmt.Sprintln(s) = %q, want %q", got, expected)
		}
	})
}

// TestIsEmpty verifies IsEmpty for empty and non-empty strings.
func TestIsEmpty(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		s := NewString("")
		if !s.IsEmpty() {
			t.Error("IsEmpty() = false for empty string, want true")
		}
	})

	t.Run("non-empty", func(t *testing.T) {
		s := NewString("something")
		if s.IsEmpty() {
			t.Error("IsEmpty() = true for non-empty string, want false")
		}
	})

	t.Run("whitespace-is-not-empty", func(t *testing.T) {
		s := NewString(" ")
		if s.IsEmpty() {
			t.Error("IsEmpty() = true for whitespace string, want false")
		}
	})
}

// TestHashMD5 verifies that HashMD5 returns the correct MD5 hex digest.
func TestHashMD5(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple", "hello"},
		{"empty", ""},
		{"password", "super-secret-password"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewString(tt.input)
			expected := md5Hex(tt.input)
			if got := s.HashMD5(); got != expected {
				t.Errorf("HashMD5() = %q, want %q", got, expected)
			}
		})
	}
}

// TestValue verifies that Value() returns the actual underlying string and is
// the ONLY way to retrieve it.
func TestValue(t *testing.T) {
	input := "the-real-secret"
	s := NewString(input)

	got := s.Value()
	if got != input {
		t.Errorf("Value() = %q, want %q", got, input)
	}
}

// TestReveal verifies that Reveal() returns the underlying plaintext as a
// typed string, mirroring Value() for call sites that statically hold a String.
func TestReveal(t *testing.T) {
	tests := []struct {
		name string
		give string
	}{
		{"non-empty", "the-real-secret"},
		{"empty", ""},
		{"unicode", "héllo wörld 🔑"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewString(tt.give).Reveal(); got != tt.give {
				t.Errorf("Reveal() = %q, want %q", got, tt.give)
			}
		})
	}
}

// TestValueReturnsString verifies that Value() returns a value of type string.
func TestValueReturnsString(t *testing.T) {
	s := NewString("test")
	v := s.Value()

	if _, ok := v.(string); !ok {
		t.Errorf("Value() returned type %T, want string", v)
	}
}

// TestMarshalJSON verifies that JSON marshaling produces "***", not the real value.
func TestMarshalJSON(t *testing.T) {
	s := NewString("json-secret")

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	// json.Marshal wraps strings in quotes: "***"
	expected := `"` + obfuscated + `"`
	if string(data) != expected {
		t.Errorf("json.Marshal() = %s, want %s", string(data), expected)
	}
}

// TestMarshalJSONInStruct verifies JSON marshaling when String is embedded in
// another struct.
func TestMarshalJSONInStruct(t *testing.T) {
	type Config struct {
		Password String `json:"password"`
	}

	c := Config{Password: NewString("embedded-secret")}
	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("json.Marshal(struct) error: %v", err)
	}

	expected := `{"password":"***"}`
	if string(data) != expected {
		t.Errorf("json.Marshal(struct) = %s, want %s", string(data), expected)
	}
}

// TestMarshalText verifies that encoding.TextMarshaler returns "***".
func TestMarshalText(t *testing.T) {
	s := NewString("text-secret")

	data, err := s.MarshalText()
	if err != nil {
		t.Fatalf("MarshalText() error: %v", err)
	}

	if string(data) != obfuscated {
		t.Errorf("MarshalText() = %q, want %q", string(data), obfuscated)
	}
}

// TestStringIsStruct verifies that String is a struct type (not a type alias),
// making direct casting from/to string impossible.
func TestStringIsStruct(t *testing.T) {
	rt := reflect.TypeFor[String]()
	if rt.Kind() != reflect.Struct {
		t.Errorf("String kind = %v, want struct", rt.Kind())
	}
}

// TestUnexportedField verifies the struct has an unexported field, preventing
// direct field access from outside the package.
func TestUnexportedField(t *testing.T) {
	rt := reflect.TypeFor[String]()
	if rt.NumField() == 0 {
		t.Fatal("String has no fields")
	}

	field := rt.Field(0)
	if field.IsExported() {
		t.Errorf("field %q is exported, want unexported", field.Name)
	}
}

// TestZeroValueIsEmpty verifies that a zero-value String behaves correctly.
func TestZeroValueIsEmpty(t *testing.T) {
	var s String

	if !s.IsEmpty() {
		t.Error("zero-value String: IsEmpty() = false, want true")
	}

	if got := s.String(); got != obfuscated {
		t.Errorf("zero-value String: String() = %q, want %q", got, obfuscated)
	}

	if got := s.Value(); got != "" {
		t.Errorf("zero-value String: Value() = %q, want %q", got, "")
	}
}

// TestNoCastFromString verifies that you cannot directly cast a string to
// String. This is a compile-time guarantee enforced by the struct type, but
// we document the intent here. If String were a type alias, this test's
// existence reminds us of the design constraint.
func TestNoCastFromString(t *testing.T) {
	// The following line must NOT compile if the type is properly a struct:
	//   _ = String("secret")
	// We verify this indirectly by checking the type is a struct.
	rt := reflect.TypeFor[String]()
	if rt.ConvertibleTo(reflect.TypeFor[string]()) {
		t.Error("String is convertible to string; it should be a struct with no direct conversion")
	}
}

// TestUnmarshalJSON verifies that JSON unmarshaling correctly populates the
// hidden value from a JSON string.
func TestUnmarshalJSON(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		var s String
		err := json.Unmarshal([]byte(`"my-secret"`), &s)
		if err != nil {
			t.Fatalf("UnmarshalJSON() error: %v", err)
		}
		if s.Value() != "my-secret" {
			t.Errorf("after UnmarshalJSON: Value() = %q, want %q", s.Value(), "my-secret")
		}
	})

	t.Run("empty-string", func(t *testing.T) {
		var s String
		err := json.Unmarshal([]byte(`""`), &s)
		if err != nil {
			t.Fatalf("UnmarshalJSON() error: %v", err)
		}
		if !s.IsEmpty() {
			t.Error("after UnmarshalJSON(\"\"): IsEmpty() = false, want true")
		}
	})

	t.Run("invalid-json", func(t *testing.T) {
		var s String
		err := json.Unmarshal([]byte(`not-json`), &s)
		if err == nil {
			t.Error("UnmarshalJSON(invalid) should return error")
		}
	})

	t.Run("non-string-type", func(t *testing.T) {
		var s String
		err := json.Unmarshal([]byte(`123`), &s)
		if err == nil {
			t.Error("UnmarshalJSON(number) should return error")
		}
	})

	t.Run("string-stays-obfuscated-after-unmarshal", func(t *testing.T) {
		var s String
		_ = json.Unmarshal([]byte(`"secret-value"`), &s)
		if got := s.String(); got != obfuscated {
			t.Errorf("after UnmarshalJSON: String() = %q, want %q", got, obfuscated)
		}
	})
}

// TestUnmarshalJSONInStruct verifies JSON unmarshaling when String is a struct
// field.
func TestUnmarshalJSONInStruct(t *testing.T) {
	type Config struct {
		Password String `json:"password"`
	}

	var c Config
	err := json.Unmarshal([]byte(`{"password":"struct-secret"}`), &c)
	if err != nil {
		t.Fatalf("json.Unmarshal(struct) error: %v", err)
	}

	if c.Password.Value() != "struct-secret" {
		t.Errorf("Password.Value() = %q, want %q", c.Password.Value(), "struct-secret")
	}

	if got := c.Password.String(); got != obfuscated {
		t.Errorf("Password.String() = %q, want %q", got, obfuscated)
	}
}

// TestUnmarshalText verifies that encoding.TextUnmarshaler correctly populates
// the hidden value.
func TestUnmarshalText(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		var s String
		err := s.UnmarshalText([]byte("text-secret"))
		if err != nil {
			t.Fatalf("UnmarshalText() error: %v", err)
		}
		if s.Value() != "text-secret" {
			t.Errorf("after UnmarshalText: Value() = %q, want %q", s.Value(), "text-secret")
		}
	})

	t.Run("empty", func(t *testing.T) {
		var s String
		err := s.UnmarshalText([]byte(""))
		if err != nil {
			t.Fatalf("UnmarshalText() error: %v", err)
		}
		if !s.IsEmpty() {
			t.Error("after UnmarshalText(\"\"): IsEmpty() = false, want true")
		}
	})

	t.Run("stays-obfuscated", func(t *testing.T) {
		var s String
		_ = s.UnmarshalText([]byte("hidden"))
		if got := s.String(); got != obfuscated {
			t.Errorf("after UnmarshalText: String() = %q, want %q", got, obfuscated)
		}
	})
}

// TestTypeForInterfaces verifies that String implements the expected interfaces
// using reflect.TypeFor (Go 1.22+).
func TestTypeForInterfaces(t *testing.T) {
	stringType := reflect.TypeFor[String]()

	tests := []struct {
		name      string
		iface     reflect.Type
		ptrNeeded bool
	}{
		{"fmt.Stringer", reflect.TypeFor[fmt.Stringer](), false},
		{"fmt.GoStringer", reflect.TypeFor[fmt.GoStringer](), false},
		{"fmt.Formatter", reflect.TypeFor[fmt.Formatter](), false},
		{"json.Marshaler", reflect.TypeFor[json.Marshaler](), false},
		{"json.Unmarshaler", reflect.TypeFor[json.Unmarshaler](), true},
		{"Hider", reflect.TypeFor[Hider](), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			check := stringType
			if tt.ptrNeeded {
				check = reflect.PointerTo(stringType)
			}
			if !check.Implements(tt.iface) {
				t.Errorf("String does not implement %s", tt.name)
			}
		})
	}
}

// md5Hex is a test helper that computes the MD5 hex digest of a string.
func md5Hex(s string) string {
	hash := md5.Sum([]byte(s))
	return hex.EncodeToString(hash[:])
}
