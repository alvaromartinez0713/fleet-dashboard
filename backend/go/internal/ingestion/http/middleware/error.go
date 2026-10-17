package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/fleet-platform/backend/internal/ingestion/domain"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// ProblemDetails represents an RFC 9457 error response.
// All field names are in English per api-standards.
type ProblemDetails struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Instance string `json:"instance,omitempty"`
}

const (
	problemContentType = "application/problem+json"
	aboutBlank         = "about:blank"
)

// RecoveryMiddleware recovers from panics and returns a 500 Problem Details response.
// Stack traces are logged but never included in the response.
func RecoveryMiddleware(logger ports.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// Extract trace ID for logging
				traceID, _ := domain.TraceIDFromContext(c.Request.Context())

				// Log the panic with stack trace (internal only)
				stackTrace := string(debug.Stack())
				logger.Error("panic recovered",
					ports.Field{Key: "trace_id", Value: string(traceID)},
					ports.Field{Key: "error", Value: fmt.Sprintf("%v", err)},
					ports.Field{Key: "stack_trace", Value: stackTrace},
				)

				// Respond with 500 Problem Details (no stack trace exposed)
				problem := ProblemDetails{
					Type:   aboutBlank,
					Title:  "Internal Server Error",
					Status: http.StatusInternalServerError,
					Detail: "An unexpected error occurred while processing your request.",
				}

				c.Header("Content-Type", problemContentType)
				c.JSON(http.StatusInternalServerError, problem)
				c.Abort()
			}
		}()

		c.Next()
	}
}

// NoRouteHandler returns a 404 Problem Details response for routes that don't exist.
func NoRouteHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		problem := ProblemDetails{
			Type:   aboutBlank,
			Title:  "Not Found",
			Status: http.StatusNotFound,
			Detail: fmt.Sprintf("The requested route %s does not exist.", c.Request.URL.Path),
		}

		c.Header("Content-Type", problemContentType)
		c.JSON(http.StatusNotFound, problem)
	}
}

// NoMethodHandler returns a 405 Problem Details response for methods that are not allowed.
func NoMethodHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		problem := ProblemDetails{
			Type:   aboutBlank,
			Title:  "Method Not Allowed",
			Status: http.StatusMethodNotAllowed,
			Detail: fmt.Sprintf("The %s method is not allowed for the requested route.", c.Request.Method),
		}

		c.Header("Content-Type", problemContentType)
		c.JSON(http.StatusMethodNotAllowed, problem)
	}
}
