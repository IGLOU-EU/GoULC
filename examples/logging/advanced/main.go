package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"gitlab.com/iglou.eu/goulc/logging"
	"gitlab.com/iglou.eu/goulc/logging/model"
)

func main() {
	// The cancel context used to gracefull shutdown on a "critical" error
	ctx, cancel := context.WithCancel(context.Background())

	// Create log writer
	logFile, _ := os.OpenFile("app.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)

	defer logFile.Close()

	w := &model.Writer{
		Out: logFile,
	}

	// Create configuration
	cfg := &model.Config{
		Level:     "DEBUG",
		Colored:   false,
		AddSource: true,
		Cancel:    cancel,
	}

	// Get the current package directory as base path
	// This is used for source code reference
	var basePath string
	if _, f, _, ok := runtime.Caller(0); ok {
		basePath = filepath.Dir(f)
	}

	// Create a new logger with custom writer output
	logger, err := logging.NewWithWriter(basePath, w, cfg)
	if err != nil {
		panic(err)
	}

	mainLogger := logger.WithGroup("main")
	mainLogger.Info("hi, the main is running fine")

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		// Give it to a fonction that "critical fail"
		// But we have a cancel context so we not use os.Exit this time
		exemple(mainLogger.WithGroup("withCancel"))
		wg.Done()
	}()

	select {
	case <-ctx.Done():
		mainLogger.Info("Context canceled", "ctx", ctx, "error", ctx.Err())
	}
	wg.Wait()

	mainLogger.Info("A gracefull shutdown with a critical example.")

	fmt.Println("take a loot at app.log and error.log")
}

func exemple(logger *slog.Logger) {
	logger.Debug("start to be critical")
	logging.Critical(logger, "This is a demo of something goes wrong, like very wrong !", "ctx", "ctx", "error", "error")
	logger.Debug("end to be critical")
}
