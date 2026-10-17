package domain

import (
	"context"

	"github.com/google/uuid"
)

// TraceID represents a unique identifier for tracing requests across the system.
type TraceID string

// ctxKeyTraceID is a private context key for storing trace IDs.
type ctxKeyTraceID struct{}

// NewTraceID generates a new random trace ID using UUID v4.
func NewTraceID() TraceID {
	return TraceID(uuid.New().String())
}

// ContextWithTraceID returns a new context with the given trace ID attached.
func ContextWithTraceID(ctx context.Context, tid TraceID) context.Context {
	return context.WithValue(ctx, ctxKeyTraceID{}, tid)
}

// TraceIDFromContext retrieves the trace ID from the context.
// Returns ("", false) if no trace ID is present in the context.
func TraceIDFromContext(ctx context.Context) (TraceID, bool) {
	tid, ok := ctx.Value(ctxKeyTraceID{}).(TraceID)
	return tid, ok
}
