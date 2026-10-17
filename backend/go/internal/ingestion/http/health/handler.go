package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// HealthResponse represents the JSON structure returned by the health endpoint.
type HealthResponse struct {
	Status string `json:"status"`
}

// HealthHandler handles health check requests for the ingestion service.
type HealthHandler struct {
	checker ports.HealthChecker
}

// NewHealthHandler creates a new HealthHandler with the provided HealthChecker.
func NewHealthHandler(checker ports.HealthChecker) *HealthHandler {
	return &HealthHandler{
		checker: checker,
	}
}

// Handle processes health check requests by calling the checker with a 1-second timeout.
// Responds with 200 {"status":"healthy"} if the service is healthy.
// Responds with 503 {"status":"unhealthy"} if the service is unhealthy or an error occurs.
// The error detail is never included in the response body.
func (h *HealthHandler) Handle(c *gin.Context) {
	// Create a context with 1-second timeout for the health check
	ctx, cancel := context.WithTimeout(c.Request.Context(), 1*time.Second)
	defer cancel()

	// Perform the health check
	healthy, err := h.checker.Check(ctx)

	// Determine response based on health status
	if healthy && err == nil {
		c.JSON(http.StatusOK, HealthResponse{Status: "healthy"})
		return
	}

	// Service is unhealthy or an error occurred
	c.JSON(http.StatusServiceUnavailable, HealthResponse{Status: "unhealthy"})
}
