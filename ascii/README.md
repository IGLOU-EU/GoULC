# 🔤 ASCII Package

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc/ascii.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc/ascii)

A Go package for validating and checking ASCII string properties. It provides simple functions to verify ASCII compliance and detect special characters.

## 🎯 Features

- **📝 ASCII Validation:**
  - `Is` - Check for Standard ASCII (0-127) validation
  - `IsPrintable` - Check for printable ASCII characters (32-126, DEL excluded)
  - `IsExtended` - Check for valid UTF-8 made only of code points up to 0xFF (Latin-1 representable text)
  - `HasNil` - Detect null bytes

Note that `IsExtended` iterates over runes while the other functions inspect raw bytes: a raw Latin-1 byte such as `"\xe9"` is rejected as invalid UTF-8, while its UTF-8 encoding `"\xc3\xa9"` is accepted.

## 📝 Examples

Usage examples can be found in the [examples](../examples/ascii) directory.

## 📜 License

This library is licensed under the [GNU Lesser General Public License v3.0 or later (LGPL v3 or later)](http://www.gnu.org/licenses/lgpl-3.0.html).
