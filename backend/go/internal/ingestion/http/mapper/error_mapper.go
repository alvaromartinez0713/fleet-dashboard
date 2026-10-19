package mapper

import (
	"net/http"

	"github.com/fleet-platform/backend/internal/ingestion/domain"
	"github.com/fleet-platform/backend/internal/ingestion/http/dto"
)

// ErrorMapper converts domain errors to RFC 9457 Problem Details responses.
// Maps domain-layer errors to appropriate HTTP status codes and problem type URIs.
// Prevents internal error details from leaking to clients.
//
// Verifies: P7 (RFC9457), P10 (Error-Mapping), P14 (Error-No-Leaks)
type ErrorMapper struct{}

// NewErrorMapper creates a new error mapper.
func NewErrorMapper() *ErrorMapper {
	return &ErrorMapper{}
}

// ToProblemDetails converts a domain error to a Problem Details response.
// Returns:
//   - status: Appropriate HTTP status code
//   - problem: RFC 9457 ProblemDetails DTO
//
// Maps known domain errors to specific types and status codes.
// Unknown errors map to 500 Internal Server Error.
func (em *ErrorMapper) ToProblemDetails(err error, traceID string) (int, *dto.ProblemDetails) {
	if err == nil {
		return http.StatusInternalServerError, dto.NewProblemDetails(
			"about:blank",
			"Unknown error",
			http.StatusInternalServerError,
		).WithTraceID(traceID)
	}

	// Check for AppError first (wraps most domain errors)
	if appErr, ok := err.(*domain.AppError); ok {
		// AppError is a generic wrapper; try to extract the actual error
		// For validation errors from NewVehiclePosition, we treat as 422
		if appErr.Inner != nil {
			// Recursively convert the wrapped error
			return em.ToProblemDetails(appErr.Inner, traceID)
		}
		// Generic AppError -> 400
		return http.StatusBadRequest,
			dto.NewProblemDetails(
				"https://api.fleet-platform.com/problems/invalid-request",
				"Invalid request",
				http.StatusBadRequest,
			).WithDetail(appErr.Message).WithTraceID(traceID)
	}

	// Map domain errors
	// Note: Check by type name since some errors are returned by value, not pointer
	switch err.(type) {
	// Validation errors -> 422 Unprocessable Content
	case domain.InvalidVehicleIDError, *domain.InvalidVehicleIDError:
		return http.StatusUnprocessableEntity,
			dto.NewProblemDetails(
				"https://api.fleet-platform.com/problems/invalid-vehicle-id",
				"Invalid vehicle ID",
				http.StatusUnprocessableEntity,
			).WithDetail(err.Error()).WithTraceID(traceID)

	case domain.InvalidGeoPositionError, *domain.InvalidGeoPositionError:
		return http.StatusUnprocessableEntity,
			dto.NewProblemDetails(
				"https://api.fleet-platform.com/problems/invalid-geo-position",
				"Invalid geographic position",
				http.StatusUnprocessableEntity,
			).WithDetail(err.Error()).WithTraceID(traceID)

	case domain.FutureTimestampError, *domain.FutureTimestampError:
		return http.StatusUnprocessableEntity,
			dto.NewProblemDetails(
				"https://api.fleet-platform.com/problems/future-timestamp",
				"Timestamp cannot be in the future",
				http.StatusUnprocessableEntity,
			).WithDetail(err.Error()).WithTraceID(traceID)

	case domain.InvalidEventIDError, *domain.InvalidEventIDError:
		return http.StatusUnprocessableEntity,
			dto.NewProblemDetails(
				"https://api.fleet-platform.com/problems/invalid-event-id",
				"Invalid event ID",
				http.StatusUnprocessableEntity,
			).WithDetail(err.Error()).WithTraceID(traceID)

	case domain.InvalidTraceIDError, *domain.InvalidTraceIDError:
		return http.StatusBadRequest,
			dto.NewProblemDetails(
				"https://api.fleet-platform.com/problems/invalid-trace-id",
				"Invalid trace ID format",
				http.StatusBadRequest,
			).WithDetail(err.Error()).WithTraceID(traceID)

	case domain.RateLimitExceededError, *domain.RateLimitExceededError:
		return http.StatusTooManyRequests,
			dto.NewProblemDetails(
				"https://api.fleet-platform.com/problems/rate-limit-exceeded",
				"Rate limit exceeded",
				http.StatusTooManyRequests,
			).WithDetail("Maximum 12 requests per 60 seconds for this vehicle").WithTraceID(traceID)

	case domain.CircuitBreakerOpenError, *domain.CircuitBreakerOpenError:
		return http.StatusServiceUnavailable,
			dto.NewProblemDetails(
				"https://api.fleet-platform.com/problems/service-unavailable",
				"Service temporarily unavailable",
				http.StatusServiceUnavailable,
			).WithDetail("The system is temporarily unable to process requests").WithTraceID(traceID)

	// Generic error handling
	default:
		// Log the error type but don't expose it to the client
		return http.StatusInternalServerError,
			dto.NewProblemDetails(
				"about:blank",
				"Internal server error",
				http.StatusInternalServerError,
			).WithTraceID(traceID)
	}
}

// ToHTTPStatus extracts the HTTP status code from an error.
// Used for middleware to set the correct response status.
func (em *ErrorMapper) ToHTTPStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	// Use a different approach since type assertions don't work well with mixed value/pointer types
	switch err.(type) {
	case domain.InvalidVehicleIDError, *domain.InvalidVehicleIDError,
		domain.InvalidGeoPositionError, *domain.InvalidGeoPositionError,
		domain.FutureTimestampError, *domain.FutureTimestampError,
		domain.InvalidEventIDError, *domain.InvalidEventIDError:
		return http.StatusUnprocessableEntity

	case domain.InvalidTraceIDError, *domain.InvalidTraceIDError:
		return http.StatusBadRequest

	case domain.RateLimitExceededError, *domain.RateLimitExceededError:
		return http.StatusTooManyRequests

	case domain.CircuitBreakerOpenError, *domain.CircuitBreakerOpenError:
		return http.StatusServiceUnavailable

	default:
		return http.StatusInternalServerError
	}
}
