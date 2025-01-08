# 📝 Logging Package

[![Go Reference](https://pkg.go.dev/badge/gitlab.com/iglou.eu/goulc/logging.svg)](https://pkg.go.dev/gitlab.com/iglou.eu/goulc/logging)

A light and flexible logging package built on top of Go's `log/slog` that supports multiple output handlers, log levels, and framework integrations.

## Features

- Multiple log levels (DEBUG, INFO, WARN, ERROR, CRITICAL)
- Colored output option
- Source code reference
- Custom formatting
- Concurrent-safe logging
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

cfg := &model.Config{
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

log := logging.New(basePath, cfg)
log.Info("Hello, World!") // Output: myapp/handler/auth.go:42: Hello, World!

// You can also use an empty string, which will show full paths
log := logging.New("", cfg)
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
    "gitlab.com/iglou.eu/goulc/logging/model"
)

// Example: Writing logs to files
logFile, _ := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
errFile, _ := os.OpenFile("error.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

writer := &model.Writer{
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
// Or
gormLogger := &logging.GormLogger{Logger: log}

&gorm.Config{
  Logger: gormLogger,
}
```

## Configuration

The logger can be configured using the `Config` struct:

```go
type Config struct {
    Level     string // Log level (DEBUG, INFO, WARN, ERROR)
    Colored   bool   // Enable colored output
    AddSource bool   // Include source code reference in logs
}
```

## License

This package is part of GoULC and is licensed under the [GNU General Public License v3.0](https://www.gnu.org/licenses/gpl-3.0).
