package ports

import (
	"context"

	"github.com/fleet-platform/backend/internal/worker/domain"
)

// Logger defines the interface for structured logging operations.
// All log entries include at minimum: timestamp, level, message, and trace_id.
// Implementations must ensure no authentication tokens, passwords, private keys,
// or secret configuration values are included in log entries.
type Logger interface {
	// Info logs an informational message with optional structured fields.
	Info(msg string, fields ...Field)

	// Warn logs a warning message with optional structured fields.
	Warn(msg string, fields ...Field)

	// Error logs an error message with optional structured fields.
	Error(msg string, fields ...Field)

	// Fatal logs a fatal error message and terminates the process.
	// Use sparingly, only for unrecoverable errors during startup.
	Fatal(msg string, fields ...Field)

	// DPanic logs a message at DPanicLevel. In development, the logger panics;
	// in production, it logs at Error level.
	DPanic(msg string, fields ...Field)

	// Panic logs a message at PanicLevel and then panics.
	Panic(msg string, fields ...Field)

	// WithTraceID stores the trace identifier in the context for later retrieval.
	// The TraceID will be automatically included in all subsequent log entries
	// that use GetTraceID to retrieve it from the context.
	WithTraceID(ctx context.Context, tid domain.TraceID) context.Context

	// GetTraceID retrieves the trace identifier from the context.
	// Returns an empty TraceID if not present in the context.
	GetTraceID(ctx context.Context) domain.TraceID

	// WithFields returns a new Logger instance with the provided fields pre-attached.
	// All subsequent log calls on the returned logger will include these fields
	// in addition to any fields passed directly.
	WithFields(fields ...Field) Logger
}

// Field represents a structured logging field with a key-value pair.
type Field struct {
	Key   string
	Value interface{}
}

// BuildFields converts variadic key-value pairs into a slice of Field structs.
// The function expects an even number of arguments where odd-indexed arguments
// are keys (strings) and even-indexed arguments are values.
func BuildFields(pairs ...interface{}) []Field {
	if len(pairs)%2 != 0 {
		// If odd number of arguments, ignore the last one
		pairs = pairs[:len(pairs)-1]
	}

	fields := make([]Field, 0, len(pairs)/2)
	for i := 0; i < len(pairs); i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			continue
		}
		fields = append(fields, Field{
			Key:   key,
			Value: pairs[i+1],
		})
	}
	return fields
}
