# 🌐 HTTP Package

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc/http.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc/http)

A Go package providing a flexible and thread-safe HTTP client with built-in support for various authentication methods and advanced features.

## 🎯 Features

- **🔒 Authentication Support:**
  - Basic Authentication
  - Digest Authentication
  - OAuth2 Client Credentials
  - Extensible authentication interface
  - An authenticator is shared across in-flight requests and keeps the
    cross-request state (token caches), so it must be safe for
    concurrent use

- **🛠️ Client Features:**
  - Thread-safe operations
  - Secure-by-design options: the zero value of `Options` is a safe
    configuration (HTTPS enforced, TLS verified, no auth/referer
    forwarding, no redirect, hardened `DefaultTimeout` and
    `DefaultMaxBodySize`), each protection has an explicit opt-out
  - Parent-child client hierarchy
  - Safe child paths from literal segments (`NewChildSegments`)
  - Configurable redirects
  - Custom header management
  - Query parameter handling
  - Rate limiting support
  - Response body size limiting (32 MiB by default, `NoBodyLimit` to
    disable)
  - Response timing
  - Context cancellation

- **🔄 Request Handling:**
  - Automatic body marshaling/unmarshaling
  - Typed access to unmarshaled responses (`Result[T]`)
  - Customizable timeout settings (35s by default, `NoTimeout` to disable)
  - TLS configuration
  - Context support
  - Redirect chain tracking

## 📦 Packages

- `http/client`: the HTTP client itself
- `http/client/auth`: authentication providers (Basic, Digest, OAuth2)
- `http/path`: URL path normalization helpers, formerly `http/utils`. The
  `Format` function (formerly `PathFormatting`) now also collapses repeated
  slashes and resolves `.` and `..` segments.

## 🔑 Authentication providers

### Basic (`http/client/auth`)

```go
func NewBasic(userID string, password hided.String) (Basic, error)
```

`NewBasic` precomputes and caches the `Basic ...` header value. Mutating
`UserID` or `Password` afterward leaves the cached header stale, rebuild
the instance with `NewBasic` instead.

### Digest (`http/client/auth`)

`Digest.Password` is a `hided.String`, so `fmt` verbs and marshaling
cannot leak it. The constructor and every hashing step return an error:

```go
func NewDigest(
    username string, password hided.String, parameters DigestParameters,
) (Digest, error)
func (d *DigestParameters) Hash(s []byte) (string, error)
func (d *Digest) A1() (string, error)
func (d *Digest) A2(method string, body []byte) (string, error)
func (d *Digest) Response(a1, a2 string) (string, error)
```

`Header` validates the parameters before signing and can return:

- `ErrNoCNonce`: QOP is set without a client nonce
- `ErrNoNC`: QOP is set without a nonce count
- `ErrUnknownQOP`: the QOP value is not supported
- `ErrUnknownAlgorithm`: the hash algorithm is not supported

### OAuth2 Client Credentials (`http/client/auth/oauth2`)

- `TokenResponse.ExpiresIn` is an `int64` lifetime in seconds, as
  defined by RFC 6749 §5.1. It was formerly a `duration.Duration`
  decoded as nanoseconds, which made tokens expire instantly.
- `Response` now exposes the named fields `TokenResponse` and
  `ErrorResponse` instead of embedding both types.
- `ErrorResponse.IsEmpty` reports whether the server returned an error,
  keying on the required `error` code of RFC 6749 §5.2.
- The error variables follow the Go convention: `ErrUnexpectedStatusCode`,
  `ErrEmptyBody`, and `ErrNoToken` (formerly `Error...` names).
- The token cache is safe for concurrent use, and tokens are refreshed
  30 seconds before expiry so an aging token is never sent in flight.

## 📝 Examples

Usage examples can be found in the [examples](../examples/http) directory.   

## 📜 License

This library is licensed under the [GNU Lesser General Public License v3.0 or later (LGPL v3 or later)](http://www.gnu.org/licenses/lgpl-3.0.html).
