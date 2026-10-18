package dto

import (
	"time"
)

// PositionResponse represents the HTTP response body after successful position ingestion (202 Accepted).
// Contains confirmation that the request was accepted for processing.
type PositionResponse struct {
	EventID   string    `json:"event_id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	VehicleID string    `json:"vehicle_id"`
	TraceID   string    `json:"trace_id"`
	ReceivedAt time.Time `json:"received_at"`
}

// ProblemDetails implements RFC 9457 Problem Details for HTTP APIs.
// Used for all error responses (4xx, 5xx).
// See: https://www.rfc-editor.org/rfc/rfc9457.html
type ProblemDetails struct {
	// Type is a URI identifying the problem type. Required.
	// Example: "https://api.fleet-platform.com/problems/validation-error"
	Type string `json:"type"`

	// Title is a short, human-readable summary of the problem. Required.
	// Should be the same for all instances of the same problem type.
	// Example: "The request contains invalid data"
	Title string `json:"title"`

	// Status is the HTTP status code for this occurrence. Required.
	// Example: 422
	Status int `json:"status"`

	// Detail is a human-readable explanation specific to this occurrence.
	// Optional but recommended.
	// Example: "latitude must be between -90 and 90"
	Detail string `json:"detail,omitempty"`

	// Instance is a URI identifying this specific occurrence of the problem.
	// Optional. Useful for linking to logs or error records.
	// Example: "/problems/occurrences/abc-123-def"
	Instance string `json:"instance,omitempty"`

	// TraceID is an internal extension for request correlation (not standard RFC 9457).
	// Allows tracing this error through logs and monitoring.
	// Example: "550e8400-e29b-41d4-a716-446655440000"
	TraceID string `json:"trace_id,omitempty"`
}

// NewProblemDetails creates a new ProblemDetails response.
func NewProblemDetails(typeURL, title string, status int) *ProblemDetails {
	return &ProblemDetails{
		Type:   typeURL,
		Title:  title,
		Status: status,
	}
}

// WithDetail adds a detail message to the problem.
func (pd *ProblemDetails) WithDetail(detail string) *ProblemDetails {
	pd.Detail = detail
	return pd
}

// WithInstance adds an instance URI to the problem.
func (pd *ProblemDetails) WithInstance(instance string) *ProblemDetails {
	pd.Instance = instance
	return pd
}

// WithTraceID adds a trace ID for request correlation.
func (pd *ProblemDetails) WithTraceID(traceID string) *ProblemDetails {
	pd.TraceID = traceID
	return pd
}
