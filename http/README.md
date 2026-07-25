# 🌐 HTTP Package

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc/http.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc/http)

A Go package providing a flexible and thread-safe HTTP client with built-in support for various authentication methods and advanced features.

## 🎯 Features

- **🔒 Authentication Support:**
  - Basic Authentication
  - Digest Authentication
  - OAuth2 Client Credentials
  - Extensible authentication interface

- **🛠️ Client Features:**
  - Thread-safe operations
  - Parent-child client hierarchy
  - Configurable redirects
  - Custom header management
  - Query parameter handling
  - Error rate tracking
  - Rate limiting support
  - Response timing
  - Context cancellation

- **🔄 Request Handling:**
  - Automatic body marshaling/unmarshaling
  - Customizable timeout settings
  - TLS configuration
  - Context support
  - Redirect chain tracking

## 📦 Packages

- `http/client`: the HTTP client itself
- `http/client/auth`: authentication providers (Basic, Digest, OAuth2)
- `http/path`: URL path normalization helpers, formerly `http/utils`. The
  `Format` function (formerly `PathFormatting`) now also collapses repeated
  slashes and resolves `.` and `..` segments.

## 📝 Examples

Usage examples can be found in the [examples](../examples/http) directory.   

## 📜 License

This library is licensed under the [GNU Lesser General Public License v3.0 or later (LGPL v3 or later)](http://www.gnu.org/licenses/lgpl-3.0.html).
