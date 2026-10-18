package domain

import (
	"context"

	"github.com/google/uuid"
)

// ctxKeyTraceID is a private context key for storing trace IDs.
type ctxKeyTraceID struct{}

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

// GenerateTraceID generates a new random trace ID using UUID v4.
// This is a simple generation function used by components that need
// a trace ID without validation (e.g., middleware fallback).
func GenerateTraceID() TraceID {
	return TraceID(uuid.New().String())
}
