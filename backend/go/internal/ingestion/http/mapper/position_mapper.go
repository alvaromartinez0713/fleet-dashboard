package mapper

import (
	"fmt"

	"github.com/fleet-platform/backend/internal/ingestion/application/commands"
	"github.com/fleet-platform/backend/internal/ingestion/domain"
	"github.com/fleet-platform/backend/internal/ingestion/http/dto"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// PositionMapper handles conversion between HTTP DTOs and domain objects.
// Responsibilities:
// - PositionRequest DTO → ReceivePositionCommand (domain command)
// - VehiclePosition Entity → PositionResponse DTO (HTTP response)
// Separates domain logic from HTTP layer concerns.
type PositionMapper struct {
	idGenerator ports.IDGenerator
	clock       ports.Clock
}

// NewPositionMapper creates a new position mapper with dependencies.
func NewPositionMapper(idGenerator ports.IDGenerator, clock ports.Clock) *PositionMapper {
	return &PositionMapper{
		idGenerator: idGenerator,
		clock:       clock,
	}
}

// FromRequest converts an HTTP request DTO to a domain command.
// Includes:
//   - Generating EventID (UUID v4) via IDGenerator
//   - Setting ReceivedTimestamp (current time) via Clock
//   - Generating TraceID (UUID v4) if not provided
//   - Including optional IdempotencyKey
//
// Returns error if request validation fails (e.g., invalid coordinates).
//
// Verifies: P1 (Validation), P2 (Async+202), P9 (Trace-ID)
func (pm *PositionMapper) FromRequest(
	req *dto.PositionRequest,
	traceID string,
	idempotencyKey *string,
) (*commands.ReceivePositionCommand, error) {
	if req == nil {
		return nil, fmt.Errorf("position request is nil")
	}

	now := pm.clock.Now()

	// Create VehicleID domain value object (validates invariants)
	vehicleID, err := domain.NewVehicleID(req.VehicleID)
	if err != nil {
		return nil, err
	}

	// Create GeoPosition domain value object (validates lat/lon ranges)
	geoPosition, err := domain.NewGeoPosition(req.Latitude, req.Longitude)
	if err != nil {
		return nil, err
	}

	// Create CapturedTimestamp (validates ISO 8601 format, checks not in future)
	capturedAt, err := domain.NewCapturedTimestamp(req.CapturedAt, now)
	if err != nil {
		return nil, err
	}

	// Create ReceivedTimestamp (set to current time)
	receivedAt, err := domain.NewReceivedTimestamp(now, now)
	if err != nil {
		return nil, err
	}

	// Create ProcessedTimestamp (set to current time)
	processedAt, err := domain.NewProcessedTimestamp(now, now)
	if err != nil {
		return nil, err
	}

	// Create TraceID (optional header provided or generate new)
	var traceIDPtr *string
	if traceID != "" {
		traceIDPtr = &traceID
	}
	traceIDObj, err := domain.NewTraceID(traceIDPtr, pm.idGenerator)
	if err != nil {
		return nil, err
	}

	// Create IdempotencyKey if provided (may be nil)
	var idempotencyKeyObj *domain.IdempotencyKey
	if idempotencyKey != nil {
		obj, err := domain.NewIdempotencyKey(idempotencyKey)
		if err != nil {
			return nil, err
		}
		idempotencyKeyObj = obj
	}

	// Generate EventID (UUID v4)
	eventID, err := domain.NewEventID(pm.idGenerator)
	if err != nil {
		return nil, err
	}

	return &commands.ReceivePositionCommand{
		VehicleID:      vehicleID,
		EventID:        eventID,
		GeoPosition:    geoPosition,
		CapturedAt:     capturedAt,
		ReceivedAt:     receivedAt,
		ProcessedAt:    processedAt,
		TraceID:        traceIDObj,
		IdempotencyKey: idempotencyKeyObj,
	}, nil
}

// FromEntity converts a domain VehiclePosition entity to an HTTP response DTO.
// Used after successful command execution to build the 202 response.
//
// Returns:
//   - event_id: UUID v4 string
//   - status: "accepted" (HTTP 202 semantics)
//   - timestamp: Current server time
//   - vehicle_id: Vehicle ID from entity
//   - trace_id: Trace ID for request correlation
//   - received_at: Server receive timestamp
//
// Verifies: P2 (Async+202), P4 (4-Timestamps), P9 (Trace-ID)
func (pm *PositionMapper) FromEntity(entity *domain.VehiclePosition) *dto.PositionResponse {
	if entity == nil {
		return nil
	}

	return &dto.PositionResponse{
		EventID:    entity.EventID().String(),
		Status:     "accepted",
		Timestamp:  pm.clock.Now(),
		VehicleID:  entity.VehicleID().String(),
		TraceID:    entity.TraceID().String(),
		ReceivedAt: entity.ReceivedAt().Value(),
	}
}
