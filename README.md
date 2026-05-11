# GoULC (Go Utils Library Collection) 🚀

[![Matrix](https://img.shields.io/matrix/iglou.eu%3Amatrix.org?logo=matrix&color=yellow)](https://matrix.to/#/#iglou.eu:matrix.org)
[![Go Report Card](https://goreportcard.com/badge/gitlab.com/iglou.eu/goulc)](https://goreportcard.com/report/gitlab.com/iglou.eu/goulc)
[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc)
[![coverage](https://gitlab.com/iglou.eu/goulc/badges/main/coverage.svg?job=coverage_report)](https://gitlab.com/iglou.eu/goulc/-/jobs)
[![License: LGPL-3.0 or later](https://img.shields.io/badge/License-LGPLv3_or_later-00802d.svg)](http://www.gnu.org/licenses/lgpl-3.0.html)

GoULC (pronounced as one word) is a collection of Go libraries I developed for my professional and personal usage. Born out of the desire to reduce cascading and/or redundant dependencies across projects, GoULC focuses on lightweight implementations while leveraging Go's standard library whenever possible. While primarily designed for my own use cases, feel free to use it if it fits your needs! 😊

## 📌 Why GoULC?

1. **🔗 Dependency Management**: Tired of importing numerous libraries for basic functionality.
2. **📚 Standard Library First**: Leverages Go's standard library whenever possible.
3. **🛠️ Framework Agnostic**: Core functionality works without external dependencies.
4. **🏢 Real-World Usage**: Actually used in professional environments.

## 🎯 Philosophy

- **Minimal Dependencies**: Preference for standard library implementations where possible.
- **Flexible Integration**: Use of interfaces for maximum adaptability.
- **Security-First**: Careful consideration of dependencies and their impact.
- **Build Tags**: External dependencies are isolated using build tags (e.g., `//go:build gorm`).

## 📦 Available Packages for Now

### 🔤 ASCII Package

A Go package for validating and checking ASCII string properties. It provides simple functions to verify ASCII compliance and detect special characters.

See the [ascii package documentation](ascii/README.md).

### 📏 ByteSize

A package for working with byte sizes in Go. It provides support for parsing, formatting, and arithmetic operations for byte sizes from Bytes up to Pebibytes.

See the [bytesize package documentation](bytesize/README.md).

### ⏱️ Duration

A Go package for handling time durations with JSON support. It wraps the standard `time.Duration` type to provide parsing and formatting capabilities for JSON serialization and deserialization.

See the [duration package documentation](duration/README.md).

### 🙈 Hided

A Go package to prevent sensitive data leakage from logs and error messages. It provides a simple interface with support for obfuscation and integration with Gorm ORM.

See the [hided package documentation](hided/README.md).

### 🌐 HTTP Package

A Go package providing a flexible and thread-safe HTTP client with built-in support for various authentication methods and advanced features.

See the [HTTP package documentation](http/README.md).

### 📄 JSONL Package

A Go package for handling JSON Lines (JSONL) data. It provides simple and efficient parsing and formatting capabilities for JSONL serialization and deserialization using generics.

See the [jsonl package documentation](jsonl/README.md).

### 📝 Logging

A light and flexible logging package built on top of `log/slog` that supports multiple output handlers, log levels, and framework integrations.

See the [logging package documentation](logging/README.md).

## 🤝 Contributing

Contributions are always welcome ! Feel free to submit a Pull Request. 🎉

Take a look at the [contributing guide](CONTRIBUTING.md)

## 📜 License

This library is licensed under the [GNU Lesser General Public License v3.0 or later (LGPL v3 or later)](http://www.gnu.org/licenses/lgpl-3.0.html).

### How to use it

The LGPLv3 license only requires that:
1. You provide attribution (include copyright notices)
2. You include a copy of the licenses with your distribution
3. Your application code remains under your chosen license

For the complete license text, see [COPYING](COPYING) AND [COPYING.LESSER](COPYING.LESSER).

### What this means

- ✅ **You can use this library in any software project** regardless of your project's license (MIT, Apache, GPL, proprietary, etc.)
- ✅ **Any modifications to this library must be shared** under the same license with the community
- ✅ **If you distribute this library** (modified or not), you must include the license and copyright notices

## 🛠️ Support

- **Report bugs** by opening an issue
- **Request features** through issues
- **Ask questions** in issues

---

Made with ❤️ by [Adrien Kara](https://gitlab.com/adrienK)