package deadletter

import (
	"context"
	"encoding/json"

	"github.com/fleet-platform/backend/internal/worker/ports"
)

// LogSink implements ports.DeadLetterSink by writing structured JSON logs.
// It logs DeadLetterRecord as JSON without including sensitive data such as
// latitude, longitude, or the complete payload.
//
// Thread-safe: holds no mutable state.
//
// RF: RF-08 (Dead-letter handling)
type LogSink struct {
	logger ports.Logger
}

// NewLogSink creates a LogSink that writes to the given logger.
func NewLogSink(logger ports.Logger) *LogSink {
	return &LogSink{
		logger: logger,
	}
}

// Write logs a dead-letter record as structured JSON.
// It always returns nil because logging failures should not block offset commits
// or prevent processing of other events. In case of logging failure, the record
// will be lost, but the offset will still be committed, which aligns with the
// requirement that dead-letter writes must not block offset commits (RF-06).
func (s *LogSink) Write(ctx context.Context, record ports.DeadLetterRecord) error {
	// Marshal the record to JSON for structured logging
	recordJSON, err := json.Marshal(record)
	if err != nil {
		// If we can't marshal the record, fall back to a simpler log message
		s.logger.Error("failed to marshal dead-letter record for logging",
			ports.BuildFields(
				"error", err.Error(),
				"trace_id", record.TraceID,
				"event_id", record.EventID,
				"vehicle_id", record.VehicleID,
				"classification", record.Classification,
				"reason_category", record.ReasonCategory,
				"occurred_at", record.OccurredAt,
			)...,
		)
		return nil // Don't fail the operation due to logging issues
	}

	// Log the dead-letter record as structured JSON
	s.logger.Warn("dead-letter record",
		ports.BuildFields(
			"trace_id", record.TraceID,
			"event_id", record.EventID,
			"vehicle_id", record.VehicleID,
			"classification", record.Classification,
			"reason_category", record.ReasonCategory,
			"occurred_at", record.OccurredAt,
			"record", string(recordJSON),
		)...,
	)

	return nil
}

// Ensure LogSink implements ports.DeadLetterSink at compile time.
var _ ports.DeadLetterSink = (*LogSink)(nil)
