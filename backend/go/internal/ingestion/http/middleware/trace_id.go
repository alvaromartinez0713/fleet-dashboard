package middleware

import (
	"regexp"

	"github.com/gin-gonic/gin"

	"github.com/fleet-platform/backend/internal/ingestion/domain"
)

// uuidV4Pattern matches the standard UUID v4 format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36 chars)
var uuidV4Pattern = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// TraceIDMiddleware extracts or generates a trace ID for each request.
// It reads the X-Trace-ID header; if the value is a valid UUID v4 (36 characters,
// format xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx) it reuses it. Otherwise, it generates
// a new trace ID using domain.NewTraceID(). The trace ID is stored in the context
// and written to the X-Trace-ID response header before calling c.Next().
func TraceIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var traceID domain.TraceID

		// Extract X-Trace-ID header from the request
		headerValue := c.GetHeader("X-Trace-ID")

		// Validate: must be exactly 36 characters and match UUID v4 format
		if len(headerValue) == 36 && uuidV4Pattern.MatchString(headerValue) {
			// Valid UUID v4 format - reuse it
			traceID = domain.TraceID(headerValue)
		} else {
			// Invalid or missing - generate a new one
			traceID = domain.NewTraceID()
		}

		// Store trace ID in context
		ctx := domain.ContextWithTraceID(c.Request.Context(), traceID)
		c.Request = c.Request.WithContext(ctx)

		// Write trace ID to response header before processing the request
		c.Header("X-Trace-ID", string(traceID))

		// Continue to next middleware/handler
		c.Next()
	}
}
