package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const IdempotencyKeyHeaderKey = "X-Idempotency-Key"
const IdempotencyKeyContextKey = "idempotency_key"

// IdempotencyKeyMiddleware validates and extracts X-Idempotency-Key header.
// Behavior:
//   - If header is absent: continues without error (optional header)
//   - If header is present: validates as UUID v4
//     - If invalid: returns 400 Bad Request
//     - If valid: stores in context
//
// The header is optional; absence does not block the request.
// This allows the application layer to decide whether to enforce idempotency.
//
// Verifies: P1 (Validation), RF-07
// Execution order: Second in middleware chain
func IdempotencyKeyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		idempotencyKey := c.GetHeader(IdempotencyKeyHeaderKey)

		// If header is absent, continue without error (optional)
		if idempotencyKey == "" {
			c.Set(IdempotencyKeyContextKey, nil)
			c.Next()
			return
		}

		// Header is present: validate as UUID v4
		if _, err := uuid.Parse(idempotencyKey); err != nil {
			// Invalid UUID format
			_ = c.Error(NewInvalidIdempotencyKeyError("invalid idempotency key format"))
			c.AbortWithStatusJSON(400, gin.H{"error": "invalid idempotency key"})
			return
		}

		// Valid: store in context
		c.Set(IdempotencyKeyContextKey, &idempotencyKey)
		c.Next()
	}
}

// GetIdempotencyKeyFromContext retrieves the idempotency key from context.
// Returns a pointer to the key string or nil if not present/invalid.
func GetIdempotencyKeyFromContext(c *gin.Context) *string {
	val, exists := c.Get(IdempotencyKeyContextKey)
	if !exists {
		return nil
	}
	key, ok := val.(*string)
	if !ok {
		return nil
	}
	return key
}

// NewInvalidIdempotencyKeyError creates an error for invalid idempotency key.
type InvalidIdempotencyKeyError struct {
	message string
}

func NewInvalidIdempotencyKeyError(msg string) *InvalidIdempotencyKeyError {
	return &InvalidIdempotencyKeyError{message: msg}
}

func (e *InvalidIdempotencyKeyError) Error() string {
	return e.message
}
