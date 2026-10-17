package middleware

import (
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fleet-platform/backend/internal/ingestion/domain"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// LoggingMiddleware creates a Gin middleware that logs HTTP request details.
// For each request, it logs: HTTP method, path, status code, latency in milliseconds,
// and the trace_id from the request context.
//
// Security constraints:
// - Does NOT log query strings (may contain sensitive parameters)
// - Does NOT log Authorization header (contains credentials)
// - Does NOT log Cookie header (contains session tokens)
// - Does NOT log request body (may contain sensitive data)
//
// The trace_id is read from the context after c.Next() completes, ensuring it
// captures the trace ID set by the TraceIDMiddleware.
func LoggingMiddleware(logger ports.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Record start time
		start := time.Now()

		// Process request
		c.Next()

		// Calculate latency in milliseconds
		latency := time.Since(start)
		latencyMs := latency.Milliseconds()

		// Extract trace ID from context
		traceID, ok := domain.TraceIDFromContext(c.Request.Context())
		if !ok {
			// Trace ID should always be present due to TraceIDMiddleware,
			// but handle the case where it's not
			traceID = ""
		}

		// Build log fields
		fields := ports.BuildFields(
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"latency_ms", latencyMs,
			"trace_id", string(traceID),
		)

		// Log the request
		logger.Info("HTTP request completed", fields...)
	}
}
