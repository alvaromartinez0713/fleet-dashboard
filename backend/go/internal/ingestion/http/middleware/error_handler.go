package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/fleet-platform/backend/internal/ingestion/http/dto"
	"github.com/fleet-platform/backend/internal/ingestion/http/mapper"
)

// ErrorHandlerMiddleware is the final middleware that catches all errors and panics.
// Responsibilities:
//   - Captures c.Error() stack from other middleware and handlers
//   - Catches panic and converts to 500 error
//   - Maps domain errors to RFC 9457 Problem Details
//   - Includes trace_id in all error responses
//   - Logs errors (for production monitoring)
//
// Verifies: P7 (RFC9457), P11 (Error-Propagation), P14 (Error-No-Leaks)
// Execution order: Last (innermost) in middleware chain - runs after all handlers
func ErrorHandlerMiddleware(errorMapper *mapper.ErrorMapper) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Defer panic recovery
		defer func() {
			if r := recover(); r != nil {
				// Panic occurred: log and convert to 500 error
				traceID := GetTraceIDFromContext(c)
				c.Header("Content-Type", "application/problem+json")
				c.JSON(http.StatusInternalServerError, dto.NewProblemDetails(
					"about:blank",
					"Internal server error",
					http.StatusInternalServerError,
				).WithDetail("An unexpected error occurred").WithTraceID(traceID))

				// Log panic (in production, this would go to structured logging)
				fmt.Printf("Panic recovered: %v\nStack: %s\n", r, debug.Stack())
			}
		}()

		// Call next handler
		c.Next()

		// After handler, check for errors
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last().Err
			traceID := GetTraceIDFromContext(c)

			// Map error to HTTP status and ProblemDetails
			status, problem := errorMapper.ToProblemDetails(err, traceID)

			// Set response content type and status
			c.Header("Content-Type", "application/problem+json")
			c.JSON(status, problem)

			// Don't call c.JSON again (already done)
			c.Abort()
		}
	}
}
