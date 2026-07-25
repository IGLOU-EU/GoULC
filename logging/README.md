# 📝 Logging Package

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc/logging.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc/logging)

A light and flexible logging package built on top of Go's `log/slog` that supports multiple output handlers, log levels, and framework integrations.

## Features

- Multiple log levels (DEBUG, INFO, WARN, ERROR, CRITICAL)
- Colored output option
- Syslog support (automatic detection under systemd journal)
- Source code reference
- Custom formatting
- Custom time format support
- Concurrent-safe logging, including between derived handlers
- Secret redaction through `ReplaceAttr` and `slog.LogValuer`
- Control-character escaping against log injection
- `sync.Pool` buffer reuse for performance
- Framework integrations (via build tags):
  - GORM (database query logging)

## Basic Usage

```bash
go get gitlab.com/iglou.eu/goulc/logging
```

```go
import (
    "path/filepath"
    "runtime"
    "gitlab.com/iglou.eu/goulc/logging"
)

cfg := &logging.Config{
    Level:     "INFO",
    Colored:   true,
    AddSource: true,
}

// Get the current package directory as base path
// This will make source references relative to this directory
var basePath string
if _, f, _, ok := runtime.Caller(0); ok {
    basePath = filepath.Dir(f)
}

log, err := logging.New(basePath, cfg)
if err != nil {
    panic(err)
}
log.Info("Hello, World!") // Output: myapp/handler/auth.go:42: Hello, World!

// You can also use an empty string, which will show full paths
log, err = logging.New("", cfg)
if err != nil {
    panic(err)
}
log.Info("Hello, World!") // Output: /home/user/projects/myapp/handler/auth.go:42: Hello, World!
```

## Custom Writer Usage

The package provides a `Writer` struct that allows you to specify custom output destinations for regular logs and error logs like so:

```go
import (
    "os"
    "path/filepath"
    "runtime"
    "gitlab.com/iglou.eu/goulc/logging"
)

// Example: Writing logs to files
logFile, _ := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
errFile, _ := os.OpenFile("error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

writer := &logging.Writer{
    Out: logFile,    // Regular logs go to app.log
    Err: errFile,    // Error logs go to error.log
}

// Get the current package directory as base path
var basePath string
if _, f, _, ok := runtime.Caller(0); ok {
    basePath = filepath.Dir(f)
}

log, err := logging.NewWithWriter(basePath, writer, cfg)
if err != nil {
    panic(err)
}
```

If nil is provided as writer, the package uses `DefaultWriter` which writes regular logs to `os.Stdout` and error logs to `os.Stderr`.

## Framework Integrations

### 📊 GORM Integration

```go
gormLogger := logging.NewGormLogger(log)

&gorm.Config{
  Logger: gormLogger,
}
```

The GORM logger also implements `gorm.ParamsFilter`: SQL parameters
implementing `hided.GormHider` are masked in traces, and the parameter
slice given by GORM is never mutated.

## Configuration

The logger can be configured using the `Config` struct. `logging.Config`
and `logging.Writer` are aliases of the `logging/model` types, so the
`model` package only needs to be imported where the alias is not enough:

```go
type Config struct {
    Level       string             // Log level (DEBUG, INFO, WARN, ERROR)
    Colored     bool               // Enable colored output
    AddSource   bool               // Include source code reference in logs
    ForceSyslog bool               // Force syslog severity prefixes
    TimeFormat  string             // Custom time format layout
    Cancel      context.CancelFunc // Called by Critical instead of os.Exit(1)
}
```

- `ForceSyslog`: Forces syslog severity prefixes (`<N>`) on each line and disables colored output. Automatically enabled when using the default writer under systemd's journal.
- `TimeFormat`: Custom time format layout following Go's reference time convention. Defaults to `[2006-01-02 15:04:05]`.

Zero-value fields are omitted when a `Config` is marshaled to JSON
(`omitzero` tags), so persisted configurations only contain what was set.

## Critical

`logging.Critical(log, msg, attrs...)` logs the message with a stack
trace, then terminates the program. If the handler carries a cancel
function (`Config.Cancel`), that function is called and your code owns
the shutdown. Otherwise `os.Exit(1)` runs immediately: deferred functions
do not run and buffers are not flushed. Reserve it for main-level failure
handling.

## Redaction and Escaping

The handler honors the two canonical slog redaction mechanisms:

- `ReplaceAttr` (in `slog.HandlerOptions`) is applied to every attribute,
  whether it comes from a log call or from `Logger.With`. Return a
  replacement attribute to rewrite a value, or a zero `slog.Attr` to drop
  it entirely.
- `slog.LogValuer` values are resolved before formatting, so a type whose
  `LogValue` returns a masked value never leaks its raw content.

```go
h := logging.NewHandler(nil, logging.DefaultWriter(), &logging.HandlerOptions{
    HandlerOptions: slog.HandlerOptions{
        ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
            if a.Key == "password" {
                return slog.String("password", "[REDACTED]")
            }
            return a
        },
    },
})
log := slog.New(h)
```

Attribute keys, textual values, and the record message are escaped on
emission: any string containing a control byte (below `0x20`, or `0x7f`)
is quoted Go-style. A `\n` in a value cannot forge an extra log line and
an `ESC` byte cannot inject ANSI sequences into the operator's terminal.

The handler passes the stdlib `testing/slogtest` suite. The remaining
deviations of the custom text format (built-ins not passed to
`ReplaceAttr`, flat dotted groups with a `[G:...]` marker) are documented
in the package comment.

## Syslog

When using the default writer (nil or no `Writer` provided), the package automatically detects whether it is running under systemd's journal by checking the `JOURNAL_STREAM` environment variable. If detected, syslog severity prefixes (`<N>`) are prepended to each log line and colored output is disabled. This detection only applies when using the default writer to avoid impacting custom writers.

You can also force this behavior manually by setting `ForceSyslog: true` in the configuration.

## License

This library is licensed under the [GNU Lesser General Public License v3.0 or later (LGPL v3 or later)](http://www.gnu.org/licenses/lgpl-3.0.html).
