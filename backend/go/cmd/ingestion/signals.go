package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// setupSignalHandler creates a channel that receives SIGINT and SIGTERM.
func setupSignalHandler() chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	return sigChan
}

// waitForSignal blocks until a signal is received on the channel.
// Logs the signal received.
func waitForSignal(sigChan <-chan os.Signal, logger ports.Logger) os.Signal {
	sig := <-sigChan
	logger.Info("received shutdown signal",
		ports.Field{Key: "signal", Value: sig.String()},
	)
	return sig
}

// waitForSecondSignal listens for a second signal with a timeout.
// Returns true if a second signal was received, false if timeout expired.
func waitForSecondSignal(sigChan <-chan os.Signal, timeout time.Duration, logger ports.Logger) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case sig := <-sigChan:
		logger.Warn("received second shutdown signal, forcing immediate exit",
			ports.Field{Key: "signal", Value: sig.String()},
		)
		return true
	case <-timer.C:
		return false
	}
}
