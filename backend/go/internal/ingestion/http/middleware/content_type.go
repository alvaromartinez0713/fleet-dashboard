package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const ContentTypeHeaderKey = "Content-Type"
const AllowedContentType = "application/json"

// ContentTypeMiddleware validates that the request Content-Type is application/json.
// Rejects with HTTP 415 Unsupported Media Type if Content-Type differs.
//
// Handles:
//   - Exact match: "application/json"
//   - With charset: "application/json; charset=utf-8" (accepted)
//   - Other types: rejected with 415
//
// Verifies: P7 (RFC9457), RF-08
// Execution order: Third in middleware chain
func ContentTypeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check Content-Type header
		contentType := c.GetHeader(ContentTypeHeaderKey)

		// Allow empty for GET requests or requests without body
		if c.Request.Method == http.MethodGet || c.Request.Method == http.MethodDelete {
			c.Next()
			return
		}

		// For POST/PATCH/PUT, require application/json
		if contentType == "" || !isJSONContentType(contentType) {
			c.Header("Content-Type", "application/problem+json")
			c.JSON(http.StatusUnsupportedMediaType, gin.H{
				"type":   "https://api.fleet-platform.com/problems/unsupported-media-type",
				"title":  "Unsupported Media Type",
				"status": http.StatusUnsupportedMediaType,
				"detail": "Content-Type must be application/json",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// isJSONContentType checks if the content type is application/json (with optional charset).
func isJSONContentType(contentType string) bool {
	// Handle "application/json" and "application/json; charset=utf-8"
	if len(contentType) < len(AllowedContentType) {
		return false
	}
	return contentType == AllowedContentType || contentType[:len(AllowedContentType)] == AllowedContentType
}
