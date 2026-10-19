package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const TraceIDHeaderKey = "X-Trace-ID"
const TraceIDContextKey = "trace_id"

// TraceIDMiddleware generates or validates X-Trace-ID UUID v4 header.
// If the header is present:
//   - Validates it as a valid UUID v4
//   - If invalid, returns 400 Bad Request (caught by ErrorHandlerMiddleware)
//   - If valid, stores it in context for downstream use
// If the header is absent:
//   - Generates a new UUID v4
//   - Stores it in context
// Always writes X-Trace-ID to response header before c.Next()
//
// Verifies: P9 (Trace-ID), RF-08
// Execution order: First (outermost) in middleware chain
func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(TraceIDHeaderKey)

		// If trace ID is provided, validate it as UUID v4
		if traceID != "" {
			if _, err := uuid.Parse(traceID); err != nil {
				// Invalid UUID format: set error and return early
				_ = c.Error(NewInvalidTraceIDError("invalid trace ID format"))
				c.AbortWithStatusJSON(400, gin.H{"error": "invalid trace ID"})
				return
			}
		} else {
			// Generate new trace ID if not provided
			traceID = uuid.New().String()
		}

		// Store trace ID in context for downstream use
		c.Set(TraceIDContextKey, traceID)

		// Write trace ID to response header
		c.Header(TraceIDHeaderKey, traceID)

		// Continue to next middleware
		c.Next()
	}
}

// GetTraceIDFromContext retrieves the trace ID from the gin context.
// Returns the trace ID string or an empty string if not found.
func GetTraceIDFromContext(c *gin.Context) string {
	val, exists := c.Get(TraceIDContextKey)
	if !exists {
		return ""
	}
	traceID, ok := val.(string)
	if !ok {
		return ""
	}
	return traceID
}

// NewInvalidTraceIDError creates an error for invalid trace ID.
type InvalidTraceIDError struct {
	message string
}

func NewInvalidTraceIDError(msg string) *InvalidTraceIDError {
	return &InvalidTraceIDError{message: msg}
}

func (e *InvalidTraceIDError) Error() string {
	return e.message
}
