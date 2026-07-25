# 🤝 Contract Package

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc/contract.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc/contract)

A Go package declaring named interfaces for common method conventions that the standard library leaves implicit or unexported. Declaring them once gives readable compile-time conformance assertions and reflect-free generic checks across the rest of GoULC, and any other codebase.

## 🎯 Features

- **🔌 Interfaces**:
  - `IsZeroer` for types reporting whether they hold their zero value, mirroring the unexported `encoding/json` interface behind the `omitzero` struct tag (Go 1.24+) and the `time.Time.IsZero` convention
  - `Emptier` for types reporting semantic emptiness, which may differ from the zero value (for example a non-nil but length-zero container)

- **🛠️ Utility**:
  - Compile-time conformance assertions that break the build when a signature drifts, instead of unreadable anonymous interfaces
  - Generic checks through a plain type assertion, cheaper than `reflect.Value.IsZero`
  - Leaf package with zero imports, so any package can depend on it without risking an import cycle

## 🚀 Usage

Assert conformance at compile time:

```go
var _ contract.IsZeroer = T{}
```

Check a value generically, without reflect:

```go
if z, ok := v.(contract.IsZeroer); ok && z.IsZero() {
    // skip the zero value
}
```

## 📝 Examples

Complete usage examples can be found in the [examples](/examples/contract) directory.

## 📜 License

This library is licensed under the [GNU Lesser General Public License v3.0 or later (LGPL v3 or later)](http://www.gnu.org/licenses/lgpl-3.0.html).
