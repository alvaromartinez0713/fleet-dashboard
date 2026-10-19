package main

import (
	"context"
	"os"
	"time"

	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// startServer starts the HTTP server, handles signals, and performs graceful shutdown.
// Returns the exit code (0 for clean shutdown, 1 for forced termination).
func startServer(server ports.HTTPServer, cfg ports.Config, logger ports.Logger) int {
	// Start the HTTP server (starts in a goroutine internally)
	if err := server.Start(); err != nil {
		logger.Error("failed to start HTTP server",
			ports.Field{Key: "error", Value: err.Error()},
		)
		return 1
	}

	// Set up signal channel for graceful shutdown
	sigChan := setupSignalHandler()

	// Wait for first signal
	waitForSignal(sigChan, logger)

	// Log shutdown initiated
	logger.Info("initiating graceful shutdown")

	// Perform graceful shutdown
	err := performGracefulShutdown(server, cfg.HTTP.ShutdownTimeout, logger, sigChan)
	if err != nil {
		logger.Error("graceful shutdown failed",
			ports.Field{Key: "error", Value: err.Error()},
		)
		return 1
	}

	logger.Info("graceful shutdown completed successfully")
	return 0
}

// performGracefulShutdown orchestrates the server shutdown with timeout and second signal handling.
func performGracefulShutdown(server ports.HTTPServer, timeout time.Duration, logger ports.Logger, sigChan <-chan os.Signal) error {
	// Create shutdown context with configured timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Channel to signal shutdown completion
	shutdownDone := make(chan error, 1)

	// Perform shutdown in a goroutine
	go func() {
		shutdownDone <- server.Stop(shutdownCtx)
	}()

	// Wait for either shutdown completion or second signal
	select {
	case err := <-shutdownDone:
		return err
	case sig := <-sigChan:
		logger.Warn("received second shutdown signal during graceful shutdown",
			ports.Field{Key: "signal", Value: sig.String()},
		)
		// Return a sentinel error to indicate forced termination
		return &forcedTerminationError{}
	}
}

// forcedTerminationError represents a forced shutdown due to second signal.
type forcedTerminationError struct{}

func (e *forcedTerminationError) Error() string {
	return "forced termination due to second signal"
}
