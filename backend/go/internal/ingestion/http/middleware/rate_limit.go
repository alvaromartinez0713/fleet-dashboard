package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/fleet-platform/backend/internal/ingestion/domain"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

const RateLimitContextKey = "rate_limit_exceeded"

// RateLimitMiddleware enforces per-vehicle rate limiting (12 requests per 60 seconds).
// Behavior:
//   - Extracts vehicle_id from request JSON body
//   - Checks RateLimiter for vehicle_id
//   - If allowed: continues
//   - If rate limit exceeded: returns 429 Too Many Requests
//
// Verifies: P6 (Rate-Limit), RF-13
// Execution order: Fourth in middleware chain
func RateLimitMiddleware(rateLimiter ports.RateLimiter, clock ports.Clock) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract vehicle_id from request body
		vehicleID, err := extractVehicleIDFromBody(c)
		if err != nil {
			// If extraction fails, allow request to continue
			// The handler will handle validation and return proper error
			c.Next()
			return
		}

		// Check if request is allowed
		allowed := rateLimiter.Allow(c.Request.Context(), vehicleID)
		if !allowed {
			// Rate limit exceeded
			c.Header("Content-Type", "application/problem+json")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"type":   "https://api.fleet-platform.com/problems/rate-limit-exceeded",
				"title":  "Rate limit exceeded",
				"status": http.StatusTooManyRequests,
				"detail": "Maximum 12 requests per 60 seconds for this vehicle",
			})
			c.Abort()
			return
		}

		// Set flag in context indicating rate limit check passed
		c.Set(RateLimitContextKey, false) // false means NOT exceeded

		c.Next()
	}
}

// extractVehicleIDFromBody extracts vehicle_id from the request JSON body.
// Reads the body, parses JSON, and returns the vehicle_id value.
// If extraction fails (malformed JSON, missing field), returns empty string and error.
// This is a lenient function: errors are logged but don't block the request.
func extractVehicleIDFromBody(c *gin.Context) (string, error) {
	// Only process POST requests with JSON body
	if c.Request.Method != http.MethodPost {
		return "", nil
	}

	// Read the raw body
	body, err := c.GetRawData()
	if err != nil || len(body) == 0 {
		return "", nil // Empty body is OK for rate limit check
	}

	// Parse JSON to extract vehicle_id
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", nil // Malformed JSON: let handler deal with it
	}

	// Extract vehicle_id
	vehicleIDRaw, exists := data["vehicle_id"]
	if !exists {
		return "", nil // Missing vehicle_id: let handler deal with it
	}

	vehicleID, ok := vehicleIDRaw.(string)
	if !ok {
		return "", nil // vehicle_id is not a string: let handler deal with it
	}

	return vehicleID, nil
}

// NewRateLimitExceededError creates a domain error for rate limit exceeded.
// Used by ErrorHandlerMiddleware to map to HTTP 429.
func NewRateLimitExceededError(vehicleID string) *domain.RateLimitExceededError {
	return &domain.RateLimitExceededError{VehicleID: vehicleID}
}
