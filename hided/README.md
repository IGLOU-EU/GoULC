# 🙈 Hided Package

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc/hided.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc/hided)

A Go package to prevent sensitive data leakage from logs and error messages. It provides a simple interface with support for obfuscation and integration with Gorm ORM.

## 🎯 Features

- **🔒 Obfuscation:**
  - Implements the Hider interface with:
    - `fmt.Stringer` returning "***"
    - `HashMD5` for hash-based obfuscation comparison

- **🛠️ Type Implementations:**
  - Provides a clear string type (`hided.String`).
  - Supports the Hider interface with `Value` for accessing the underlying data.

- **🔥 Gorm Integration:**
  - Custom Gorm-enabled string type (`GormString`) to use with `gorm.Valuer` and a custom `GormHider` interface.
  - Allows both clear and obfuscated representations for ORM operations.

## 📝 Examples

Usage examples can be found in the [examples](../examples/hided) directory.   
For Gorm example, view the [gorm logging examples](../examples/logging/gorm)

## 📜 License

This library is licensed under the [GNU Lesser General Public License v3.0 or later (LGPL v3 or later)](http://www.gnu.org/licenses/lgpl-3.0.html).
