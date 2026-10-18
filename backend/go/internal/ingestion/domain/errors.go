package domain

import (
	"fmt"
	"time"
)

// AppError is the base error type for the api domain layer.
type AppError struct {
	Code          string
	Message       string
	Inner         error
	IsRecoverable bool
}

// Error implements the error interface.
func (e *AppError) Error() string {
	if e.Inner != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Inner)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the wrapped error.
func (e *AppError) Unwrap() error {
	return e.Inner
}

// NewAppError creates a new non-recoverable AppError.
func NewAppError(code, message string, inner error) *AppError {
	return &AppError{
		Code:          code,
		Message:       message,
		Inner:         inner,
		IsRecoverable: false,
	}
}

// NewRecoverableError creates a new recoverable AppError.
func NewRecoverableError(code, message string, inner error) *AppError {
	return &AppError{
		Code:          code,
		Message:       message,
		Inner:         inner,
		IsRecoverable: true,
	}
}

// ---
// Domain-Specific Errors (Value Object Validation)
// ---

// FutureTimestampError occurs when a captured timestamp is in the future.
type FutureTimestampError struct {
	Value time.Time
}

// Error implements the error interface.
func (e FutureTimestampError) Error() string {
	return fmt.Sprintf("timestamp cannot be in the future: %v", e.Value)
}

// InvalidTimestampFormatError occurs when a timestamp string is not in valid ISO 8601 UTC format.
type InvalidTimestampFormatError struct {
	Value string
}

// Error implements the error interface.
func (e InvalidTimestampFormatError) Error() string {
	return fmt.Sprintf("invalid ISO 8601 UTC timestamp format: %s", e.Value)
}

// InvalidVehicleIDError occurs when vehicle_id violates validation rules.
type InvalidVehicleIDError struct {
	Value  string
	Reason string
}

// Error implements the error interface.
func (e InvalidVehicleIDError) Error() string {
	return fmt.Sprintf("invalid vehicle_id: %s", e.Reason)
}

// InvalidGeoPositionError occurs when latitude or longitude is out of valid range.
type InvalidGeoPositionError struct {
	Latitude  *float64
	Longitude *float64
	Reason    string
}

// Error implements the error interface.
func (e InvalidGeoPositionError) Error() string {
	return fmt.Sprintf("invalid geolocation: %s", e.Reason)
}

// InvalidEventIDError occurs when event_id is not a valid UUID v4.
type InvalidEventIDError struct {
	Value string
}

// Error implements the error interface.
func (e InvalidEventIDError) Error() string {
	return "event_id must be valid UUID v4"
}

// InvalidTraceIDError occurs when X-Trace-ID header is not a valid UUID v4.
type InvalidTraceIDError struct {
	Value string
}

// Error implements the error interface.
func (e InvalidTraceIDError) Error() string {
	return "X-Trace-ID header must be valid UUID v4"
}

// InvalidIdempotencyKeyError occurs when X-Idempotency-Key header is not a valid UUID v4.
type InvalidIdempotencyKeyError struct {
	Value string
}

// Error implements the error interface.
func (e InvalidIdempotencyKeyError) Error() string {
	return "X-Idempotency-Key header must be valid UUID v4"
}

// RateLimitExceededError occurs when a vehicle exceeds the rate limit (12 requests per 60s).
type RateLimitExceededError struct {
	VehicleID string
}

// Error implements the error interface.
func (e RateLimitExceededError) Error() string {
	return fmt.Sprintf("rate limit exceeded for vehicle: %s (max 12 requests per 60 seconds)", e.VehicleID)
}

// CircuitBreakerOpenError occurs when the Circuit Breaker is in OPEN state and Kafka is unavailable.
type CircuitBreakerOpenError struct{}

// Error implements the error interface.
func (e CircuitBreakerOpenError) Error() string {
	return "circuit breaker is open: kafka is currently unavailable"
}
