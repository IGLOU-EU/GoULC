# ⏱️ Duration Package

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc/duration.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc/duration)

A Go package for handling time durations with JSON support. It wraps the standard `time.Duration` type to provide parsing and formatting capabilities for JSON serialization and deserialization.

## 🎯 Features

- **🔌 Interfacing**:
  - JSON Marshaler/Unmarshaler for duration values
  - Text Marshaler/Unmarshaler, enabling use as JSON map keys or with `flag.TextVar`
  - `IsZero` support for the JSON `omitzero` tag option (Go 1.24+)
  - Seamless integration with Go's `time.Duration`
  - Support for multiple input formats

- **🔄 Input Formats**:
  - Parse string representations using `time.ParseDuration` format
  - Parse bare JSON numbers as nanosecond counts, consistently with `time.Duration`. The number must be a whole number fitting in an int64: fractions, scientific notation, and out-of-range values are rejected with an error instead of losing precision or silently wrapping around
  - Treat JSON `null` as a no-op, following the `encoding/json` convention
  - Automatic type detection during JSON unmarshaling

- **🛠️ Utility**:
  - Full compatibility with standard `time.Duration` functionality
  - Convert to/from standard `time.Duration` with `New` and `ToTimeDuration`
  - Maintain all arithmetic and comparison capabilities
  - Preserve duration precision

## ⚠️ Important Notes

- Failures are reported through the exported sentinel errors `ErrBadDuration` (wraps any parse failure) and `ErrDurationInvalidType` (JSON value of an unsupported type), match them with `errors.Is`

## 📝 Examples

Complete usage examples can be found in the [examples](/examples/duration) directory.

## 📜 License

This library is licensed under the [GNU Lesser General Public License v3.0 or later (LGPL v3 or later)](http://www.gnu.org/licenses/lgpl-3.0.html).
