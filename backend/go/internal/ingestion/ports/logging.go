package ports

import (
	"context"
)

// Logger defines the contract for structured logging throughout the ingestion service.
// Implementations must emit JSON-formatted log entries with at least timestamp, level,
// message, and trace_id fields. The logger must never include authentication tokens,
// passwords, private keys, or secret configuration values in any log entry.
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
	// tid is a string representation of the trace ID.
	WithTraceID(ctx context.Context, tid string) context.Context

	// GetTraceID retrieves the trace identifier from the context.
	// Returns an empty string if not present in the context.
	GetTraceID(ctx context.Context) string

	// WithFields returns a new Logger instance with the provided fields
	// pre-attached. All subsequent log calls on the returned logger will
	// include these fields in addition to any fields passed directly.
	WithFields(fields ...Field) Logger
}

// Field represents a structured key-value pair for logging.
// The Value can be any JSON-serializable type.
type Field struct {
	Key   string
	Value interface{}
}

// BuildFields constructs a slice of Field from variadic pairs of arguments.
// Each pair consists of a key (string) followed by a value (any type).
// If the number of arguments is odd, the last argument is ignored.
//
// Example:
//
//	fields := BuildFields("user_id", 42, "action", "login", "success", true)
//	// Returns: []Field{
//	//   {Key: "user_id", Value: 42},
//	//   {Key: "action", Value: "login"},
//	//   {Key: "success", Value: true},
//	// }
func BuildFields(pairs ...interface{}) []Field {
	if len(pairs) == 0 {
		return nil
	}

	// Ensure we have pairs (even number of arguments)
	count := len(pairs) / 2
	fields := make([]Field, 0, count)

	for i := 0; i < count*2; i += 2 {
		key, ok := pairs[i].(string)
		if !ok {
			// Skip invalid pairs where the key is not a string
			continue
		}
		fields = append(fields, Field{
			Key:   key,
			Value: pairs[i+1],
		})
	}

	return fields
}
