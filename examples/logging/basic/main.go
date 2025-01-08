package main

import (
	"errors"
	"log/slog"
	"time"

	"gitlab.com/iglou.eu/goulc/logging"
	"gitlab.com/iglou.eu/goulc/logging/model"
)

func main() {
	// Create configuration
	cfg := &model.Config{
		Level:     "DEBUG",
		Colored:   true,
		AddSource: true,
	}

	// Create a new logger with configuration
	// ignore basePath to show full file path
	logger, err := logging.New("", cfg)
	if err != nil {
		panic(err)
	}

	// Basic logging
	logger.Debug("This is a debug message")
	logger.Info("This is an info message")
	logger.Warn("This is a warning message")
	logger.Error("This is an error message")

	// Logging with namespaces
	mainLogger := logger.WithGroup("main")
	mainLogger.Info("Starting application")

	// Logging with persistent attributes
	user := mainLogger.WithGroup("user").With("user", "john", "type", "account")
	user.Info("User logged in", "action", "loged in")
	user.Info("User logged out", "action", "loged out")

	// Using groups into persistent attributes
	dbLogger := mainLogger.WithGroup("db").With(
		slog.Group("database", "host", "localhost", "port", 5432),
		"group", slog.GroupValue(slog.String("driver", "postgres"), slog.String("username", "postgres"), slog.String("password", "secret")),
	)
	dbLogger.Info("Connected to database")
	dbLogger.With("query", "SELECT * FROM users").Info("Executing query")

	// Error handling
	err = errors.New("operation failed")
	user.WithGroup("insert").Error("Operation failed", "an error occurred", err)

	// Critical error
	go func() {
		logging.Critical(dbLogger.WithGroup("create"), "This is a demo of something goes wrong, like very wrong !")
	}()

	time.Sleep(time.Second)
	logger.Info("This point will never be reached")
}
