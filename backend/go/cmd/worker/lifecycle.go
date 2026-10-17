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

	// NOTE: Message consumption logic (Kafka consumer, message handlers, processing pipeline)
	// will be implemented in the ingestion spec. This bootstrap implementation sets up
	// the service lifecycle and graceful shutdown handling, with the actual business logic
	// to be added when Kafka and PostgreSQL adapters are integrated.

	logger.Info("worker service ready")

	// Set up signal channel for graceful shutdown
	sigChan := setupSignalHandler()

	// Wait for first signal
	waitForSignal(sigChan, logger)

	// Log shutdown initiated
	logger.Info("initiating graceful shutdown")

	// Perform cleanup with timeout
	// As per requirement 7.4, the worker must complete shutdown within 10 seconds hard limit
	err := performCleanupWithTimeout(cleanup, 10*time.Second, logger, sigChan)
	if err != nil {
		logger.Error("cleanup failed",
			ports.Field{Key: "error", Value: err.Error()},
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
		// NOTE: In the ingestion spec, cleanup will include:
		// - Stop consuming from Kafka
		// - Wait for in-flight messages to complete processing (up to shutdown timeout)
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
				ports.Field{Key: "timeout", Value: timeout.String()},
			)
			return &cleanupTimeoutError{timeout: timeout}
		}
		return cleanupCtx.Err()

	case sig := <-sigChan:
		logger.Warn("received second shutdown signal during cleanup",
			ports.Field{Key: "signal", Value: sig.String()},
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
