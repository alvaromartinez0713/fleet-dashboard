package main

import (
	"context"
	"os"
	"time"

	"github.com/fleet-platform/backend/internal/worker/ports"
)

// runWorker runs the worker process, handles signals, and performs cleanup.
// Returns the exit code (0 for clean shutdown, 1 for forced termination).
func runWorker(cleanup func(), cfg ports.Config, logger ports.Logger) int {
	// Log worker started
	logger.Info("worker service starting")

	logger.Info("worker service ready")

	// Set up signal channel for graceful shutdown
	sigChan := setupSignalHandler()

	// Wait for first signal
	waitForSignal(sigChan, logger)

	// Log shutdown initiated
	logger.Info("initiating graceful shutdown")

	// Perform cleanup with timeout
	// Worker must complete shutdown with timeout (RF-01, RF-05)
	err := performCleanupWithTimeout(cleanup, 10*time.Second, logger, sigChan)
	if err != nil {
		logger.Error("cleanup failed",
			ports.BuildFields("error", err.Error())...,
		)
		return 1
	}

	logger.Info("worker service stopped")
	return 0
}

// performCleanupWithTimeout executes cleanup with a timeout and second signal handling.
func performCleanupWithTimeout(cleanup func(), timeout time.Duration, logger ports.Logger, sigChan <-chan os.Signal) error {
	// Create cleanup context with timeout
	cleanupCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Channel to signal cleanup completion
	cleanupDone := make(chan struct{})

	// Perform cleanup in a goroutine
	go func() {
		// Cleanup includes:
		// - Stop consuming from Kafka (via runner.Stop())
		// - Close database connections
		// - Release all acquired resources
		cleanup()
		close(cleanupDone)
	}()

	// Wait for either cleanup completion, timeout, or second signal
	select {
	case <-cleanupDone:
		logger.Info("cleanup completed successfully")
		return nil

	case <-cleanupCtx.Done():
		if cleanupCtx.Err() == context.DeadlineExceeded {
			logger.Warn("cleanup timeout exceeded",
				ports.BuildFields("timeout", timeout.String())...,
			)
			return &cleanupTimeoutError{timeout: timeout}
		}
		return cleanupCtx.Err()

	case sig := <-sigChan:
		logger.Warn("received second shutdown signal during cleanup",
			ports.BuildFields("signal", sig.String())...,
		)
		return &forcedTerminationError{}
	}
}

// cleanupTimeoutError represents a cleanup timeout.
type cleanupTimeoutError struct {
	timeout time.Duration
}

func (e *cleanupTimeoutError) Error() string {
	return "cleanup timeout exceeded: " + e.timeout.String()
}

// forcedTerminationError represents a forced shutdown due to second signal.
type forcedTerminationError struct{}

func (e *forcedTerminationError) Error() string {
	return "forced termination due to second signal"
}
