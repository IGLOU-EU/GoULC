# ⏱️ Duration Package

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc/duration.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc/duration)

A Go package for handling time durations with JSON support. It wraps the standard `time.Duration` type to provide parsing and formatting capabilities for JSON serialization and deserialization.

## 🎯 Features

- **🔌 Interfacing**:
  - JSON Marshaler/Unmarshaler for duration values
  - Seamless integration with Go's `time.Duration`
  - Support for multiple input formats

- **🔄 Input Formats**:
  - Parse numeric values (integers and floats) as nanoseconds
  - Parse string representations using `time.ParseDuration` format
  - Automatic type detection during JSON unmarshaling

- **🛠️ Utility**:
  - Full compatibility with standard `time.Duration` functionality
  - Convert to/from standard `time.Duration`
  - Maintain all arithmetic and comparison capabilities
  - Preserve duration precision

## 📝 Examples

Complete usage examples can be found in the [examples](/examples/duration) directory.

## 📜 License

This library is licensed under the [GNU Lesser General Public License v3.0 or later (LGPL v3 or later)](http://www.gnu.org/licenses/lgpl-3.0.html).
