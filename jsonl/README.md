# 📄 JSONL Package

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc/jsonl.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc/jsonl)

A Go package for handling JSON Lines (JSONL) data. It provides simple and efficient parsing and formatting capabilities for JSONL serialization and deserialization using generics.

## 🎯 Features

- **🔌 Interfacing**:
  - `Marshal[T any]` and `Unmarshal[T any]` functions using Go Generics
  - `UnmarshalStream[T any]` for memory-efficient parsing from any `io.Reader`
  - Seamless integration with standard `encoding/json`
  - Compliant with the [JSON Lines specification](https://jsonlines.org/)

- **🔄 Capabilities**:
  - Safely encode slices of any type into newline-separated JSON objects
  - Gracefully decode JSONL data streams into typed slices
  - Properly handles line terminators and ignores empty trailing lines

- **🛠️ Utility**:
  - Fast execution by pre-allocating buffer capacities
  - Strict validation against blank lines to respect JSONL specs
  - Returns `nil` without error for empty inputs or empty slices

## 📝 Examples

Complete usage examples can be found in the [examples](/examples/jsonl) directory.

## 📜 License

This library is licensed under the [GNU Lesser General Public License v3.0 or later (LGPL v3 or later)](http://www.gnu.org/licenses/lgpl-3.0.html).
