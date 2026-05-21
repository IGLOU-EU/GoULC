# 🙈 Hided Package

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc/hided.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc/hided)

A Go package to prevent sensitive data leakage from logs, error messages, and any form of string output. It provides a secure struct-based approach with support for obfuscation and integration with Gorm ORM.

## 🎯 Features

- **🔒 Secure by Design:**
  - `hided.String` is a struct with an unexported `[]byte` field, making it impossible to leak via type assertion or casting.
  - `String()`, `GoString()`, and `Format()` all return `"***"`, covering `%s`, `%v`, `%q`, `%+v`, `%#v`, and all other fmt verbs.
  - `MarshalJSON()` and `MarshalText()` return `"***"`, preventing leakage through JSON or text serialization.
  - Built-in `print()`/`println()` cannot leak the value since the underlying data is a struct with unexported fields.
  - `Value()` is the **only** way to access the real value.

- **🛠️ Type Implementations:**
  - Constructor: `NewString(s string) String` to create a hided string.
  - Implements the `Hider` interface with `Value` for accessing the underlying data.
  - `HashMD5` for hash-based obfuscation comparison.

- **🧬 Generic Accessor:**
  - `hided.Value[T](h Hider) T` returns the underlying value asserted to `T`.
  - Type-safe alternative to `h.Value().(T)`: no panic on mismatch, no
    comma-ok boilerplate. On a type mismatch the zero value of `T` is
    returned. Example: `pwd := hided.Value[string](myHidedSecret)`.

- **🔥 Gorm Integration:**
  - Custom Gorm-enabled string type (`GormString`) to use with `gorm.Valuer` and a custom `GormHider` interface.
  - Allows both clear and obfuscated representations for ORM operations.

## 📝 Examples

Usage examples can be found in the [examples](../examples/hided) directory.   
For Gorm example, view the [gorm logging examples](../examples/logging/gorm)

## 📜 License

This library is licensed under the [GNU Lesser General Public License v3.0 or later (LGPL v3 or later)](http://www.gnu.org/licenses/lgpl-3.0.html).
