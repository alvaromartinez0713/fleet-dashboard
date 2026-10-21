package parser

import (
	"encoding/json"
	"time"

	"github.com/fleet-platform/backend/internal/worker/domain"
	"github.com/fleet-platform/backend/internal/worker/ports"
)

// PositionEventParser is responsible for parsing raw JSON payloads and constructing domain entities.
// This is an adapter (inbound) that converts external format to domain entities.
//
// Responsibilities:
// - Parse JSON → ports.ConsumedPositionEvent (ACL boundary)
// - Extract primitives and call domain factory
//
// Clean Architecture: The adapter extracts primitives and calls the domain factory.
// The domain does NOT know about DTOs from outer layers.
type PositionEventParser struct{}

// NewPositionEventParser constructs the parser.
func NewPositionEventParser() *PositionEventParser {
	return &PositionEventParser{}
}

// ParseAndValidate parses raw JSON bytes and constructs a validated domain entity.
//
// Flow:
// 1. Parse JSON → ports.ConsumedPositionEvent (ACL)
// 2. Extract primitives
// 3. Call domain factory with primitives
// 4. Domain validates and constructs entity
//
// Args:
//   rawPayload: Raw JSON bytes from Kafka message
//   validationTime: Current time used for temporal validation (capturedAt, receivedAt, processedAt constraints)
//   persistenceTime: Time when this entity will be persisted (generates storedAt)
//
// Returns validated VehiclePosition entity or domain error.
func (p *PositionEventParser) ParseAndValidate(rawPayload []byte, validationTime time.Time, persistenceTime time.Time) (*domain.VehiclePosition, error) {
	// Step 1: Parse JSON into ports DTO (ACL boundary)
	var portsEvent ports.ConsumedPositionEvent
	if err := json.Unmarshal(rawPayload, &portsEvent); err != nil {
		return nil, domain.NewAppError(
			"MALFORMED_PAYLOAD",
			"failed to parse JSON payload",
			err,
		)
	}

	// Step 2: Call domain factory with primitive values
	// The domain validates and constructs value objects internally
	position, err := domain.NewVehiclePosition(
		portsEvent.EventID,
		portsEvent.VehicleID,
		portsEvent.Latitude,
		portsEvent.Longitude,
		portsEvent.Accuracy,
		portsEvent.CapturedAt,
		portsEvent.ReceivedAt,
		portsEvent.ProcessedAt,
		portsEvent.TraceID,
		portsEvent.IdempotencyKey,
		validationTime,
		persistenceTime,
	)
	if err != nil {
		return nil, err
	}

	return position, nil
}

// ExtractEventID attempts best-effort extraction of event_id from malformed payload.
// Used for dead-letter logging when parsing fails.
func (p *PositionEventParser) ExtractEventID(rawPayload []byte) string {
	var partial struct {
		EventID string `json:"event_id"`
	}
	if err := json.Unmarshal(rawPayload, &partial); err == nil {
		return partial.EventID
	}
	return ""
}

// ExtractVehicleID attempts best-effort extraction of vehicle_id from malformed payload.
// Used for dead-letter logging when parsing fails.
func (p *PositionEventParser) ExtractVehicleID(rawPayload []byte) string {
	var partial struct {
		VehicleID string `json:"vehicle_id"`
	}
	if err := json.Unmarshal(rawPayload, &partial); err == nil {
		return partial.VehicleID
	}
	return ""
}
