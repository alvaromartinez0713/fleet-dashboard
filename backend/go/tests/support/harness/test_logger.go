package support

import (
	"context"

	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// TestLogger is a simple logger implementation for use in tests.
// It implements the ports.Logger interface but doesn't actually output logs.
// This keeps tests quiet while still satisfying the interface.
type TestLogger struct{}

// NewTestLogger creates a new TestLogger.
func NewTestLogger() *TestLogger {
	return &TestLogger{}
}

// Info logs an informational message (no-op for tests).
func (t *TestLogger) Info(msg string, fields ...ports.Field) {}

// Warn logs a warning message (no-op for tests).
func (t *TestLogger) Warn(msg string, fields ...ports.Field) {}

// Error logs an error message (no-op for tests).
func (t *TestLogger) Error(msg string, fields ...ports.Field) {}

// Fatal logs a fatal error message (no-op for tests).
func (t *TestLogger) Fatal(msg string, fields ...ports.Field) {}

// DPanic logs a message at DPanicLevel (no-op for tests).
func (t *TestLogger) DPanic(msg string, fields ...ports.Field) {}

// Panic logs a message and panics (no-op for tests).
func (t *TestLogger) Panic(msg string, fields ...ports.Field) {}

// WithTraceID stores the trace identifier in context (no-op for tests).
func (t *TestLogger) WithTraceID(ctx context.Context, tid string) context.Context {
	return ctx
}

// GetTraceID retrieves the trace identifier from context (no-op for tests).
func (t *TestLogger) GetTraceID(ctx context.Context) string {
	return ""
}

// WithFields returns a new logger with pre-attached fields (returns self for tests).
func (t *TestLogger) WithFields(fields ...ports.Field) ports.Logger {
	return t
}
