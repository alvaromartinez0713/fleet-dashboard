package messaging

import (
	"context"
	"time"

	"github.com/fleet-platform/backend/internal/worker/adapters/inbound/parser"
	"github.com/fleet-platform/backend/internal/worker/domain"
	"github.com/fleet-platform/backend/internal/worker/ports"
)

// ErrorStrategy encapsulates error handling logic for message processing.
//
// Responsibilities:
// - Classify errors (recoverable, non-recoverable, circuit breaker)
// - Decide action: dead-letter, commit, pause, retry
// - Write dead-letter records
//
// This separates error handling logic from the main runner loop.
type ErrorStrategy struct {
	parser         *parser.PositionEventParser
	deadLetterSink ports.DeadLetterSink
	clock          ports.Clock
	logger         ports.Logger
}

// NewErrorStrategy constructs the error strategy.
func NewErrorStrategy(
	parser *parser.PositionEventParser,
	deadLetterSink ports.DeadLetterSink,
	clock ports.Clock,
	logger ports.Logger,
) *ErrorStrategy {
	return &ErrorStrategy{
		parser:         parser,
		deadLetterSink: deadLetterSink,
		clock:          clock,
		logger:         logger,
	}
}

// ErrorAction represents the decision after handling an error.
type ErrorAction struct {
	ShouldCommit bool          // Should commit Kafka offset
	ShouldPause  bool          // Should pause consumption (circuit breaker open)
	PauseDuration time.Duration // How long to pause
}

// HandleParseError handles errors from parsing/validation.
// Returns whether to commit the offset.
func (es *ErrorStrategy) HandleParseError(
	ctx context.Context,
	err error,
	traceID domain.TraceID,
	rawPayload []byte,
) ErrorAction {
	appErr, ok := err.(*domain.AppError)
	if !ok {
		es.logger.Error("unexpected parse error type",
			ports.Field{Key: "trace_id", Value: traceID.String()},
			ports.Field{Key: "error", Value: err.Error()},
		)
		// Unknown error - don't commit, let it retry
		return ErrorAction{ShouldCommit: false, ShouldPause: false}
	}

	es.logger.Error("failed to parse/validate payload",
		ports.Field{Key: "trace_id", Value: traceID.String()},
		ports.Field{Key: "error_code", Value: appErr.Code},
	)

	// Dead-letter (best-effort extraction of IDs)
	dl := ports.DeadLetterRecord{
		TraceID:        traceID.String(),
		EventID:        es.parser.ExtractEventID(rawPayload),
		VehicleID:      es.parser.ExtractVehicleID(rawPayload),
		Classification: "non_recoverable",
		ReasonCategory: appErr.Code,
		OccurredAt:     es.clock.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"),
	}

	if dlErr := es.deadLetterSink.Write(ctx, dl); dlErr != nil {
		es.logger.Error("failed to write dead-letter",
			ports.Field{Key: "trace_id", Value: traceID.String()},
		)
		// Don't commit if dead-letter failed - we can't lose the event
		return ErrorAction{ShouldCommit: false, ShouldPause: false}
	}

	// Dead-letter succeeded - commit offset (event is non-recoverable)
	return ErrorAction{ShouldCommit: true, ShouldPause: false}
}

// HandleProcessorError handles errors from the processor (use case + persistence).
// Returns whether to commit the offset and whether to pause consumption.
func (es *ErrorStrategy) HandleProcessorError(
	ctx context.Context,
	err error,
	traceID domain.TraceID,
	position *domain.VehiclePosition,
) ErrorAction {
	appErr, ok := err.(*domain.AppError)
	if !ok {
		es.logger.Error("unexpected processor error type",
			ports.Field{Key: "trace_id", Value: traceID.String()},
			ports.Field{Key: "error", Value: err.Error()},
		)
		// Unknown error - don't commit
		return ErrorAction{ShouldCommit: false, ShouldPause: false}
	}

	// Circuit breaker opened - pause consumption, don't commit
	if appErr.Code == "CIRCUIT_BREAKER_OPEN" {
		es.logger.Warn("circuit breaker opened, pausing consumption",
			ports.Field{Key: "trace_id", Value: traceID.String()},
		)
		return ErrorAction{
			ShouldCommit:  false,
			ShouldPause:   true,
			PauseDuration: 5 * time.Second,
		}
	}

	// Retries exhausted - dead-letter and commit
	if appErr.Code == "RETRIES_EXHAUSTED" {
		es.logger.Error("retries exhausted for event",
			ports.Field{Key: "trace_id", Value: traceID.String()},
			ports.Field{Key: "event_id", Value: position.EventID().String()},
		)
		es.deadLetterPosition(ctx, traceID, position, "recoverable", "RETRIES_EXHAUSTED")
		return ErrorAction{ShouldCommit: true, ShouldPause: false}
	}

	// Other errors (validation, non-recoverable persistence) - dead-letter and commit
	es.logger.Error("processor failed",
		ports.Field{Key: "trace_id", Value: traceID.String()},
		ports.Field{Key: "event_id", Value: position.EventID().String()},
		ports.Field{Key: "error_code", Value: appErr.Code},
	)
	es.deadLetterPosition(ctx, traceID, position, "non_recoverable", appErr.Code)
	return ErrorAction{ShouldCommit: true, ShouldPause: false}
}

func (es *ErrorStrategy) deadLetterPosition(
	ctx context.Context,
	traceID domain.TraceID,
	position *domain.VehiclePosition,
	classification, reason string,
) {
	vehicleID := ""
	if position.VehicleID() != nil {
		vehicleID = position.VehicleID().String()
	}

	dl := ports.DeadLetterRecord{
		TraceID:        traceID.String(),
		EventID:        position.EventID().String(),
		VehicleID:      vehicleID,
		Classification: classification,
		ReasonCategory: reason,
		OccurredAt:     es.clock.Now().UTC().Format("2006-01-02T15:04:05.000Z07:00"),
	}

	if err := es.deadLetterSink.Write(ctx, dl); err != nil {
		es.logger.Error("failed to write dead-letter",
			ports.Field{Key: "trace_id", Value: traceID.String()},
			ports.Field{Key: "event_id", Value: position.EventID().String()},
		)
	}
}
