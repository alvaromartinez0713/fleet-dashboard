package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fleet-platform/backend/internal/worker/config"
	"github.com/fleet-platform/backend/internal/worker/ports"
)

// Run bootstraps the worker.
func Run() int {
	// Load config
	cfg, err := config.NewEnvConfigLoader().Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config load: %v\n", err)
		return 1
	}

	// Validate config
	if errs := config.NewConfigValidator().Validate(cfg); len(errs) > 0 {
		fmt.Fprintf(os.Stderr, "validation failed:\n")
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "  %s: %s\n", e.Field, e.Message)
		}
		return 1
	}

	// Setup logger
	logger, logCleanup, err := config.NewLogger(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "logger: %v\n", err)
		return 1
	}
	defer logCleanup()

	// Dependency Injection (cmd layer responsibility)
	stack, err := InjectDependencies(cfg, logger)
	if err != nil {
		logger.Error("inject failed", ports.BuildFields("error", err.Error())...)
		return 1
	}
	defer stack.Cleanup()

	// Start runner
	if err := stack.Runner.Start(context.Background()); err != nil {
		logger.Error("start failed", ports.BuildFields("error", err.Error())...)
		return 1
	}

	// Lifecycle
	return runWorker(stack.Cleanup, cfg, logger)
}
