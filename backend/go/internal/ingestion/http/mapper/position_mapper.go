package mapper

import (
	"fmt"
	"time"

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

	// Create CapturedTimestamp (validates not in future)
	capturedAt, err := domain.NewCapturedTimestamp(req.CapturedAt, pm.clock.Now())
	if err != nil {
		return nil, err
	}

	// Create ReceivedTimestamp (set to current time)
	receivedAt, err := domain.NewReceivedTimestamp(pm.clock.Now())
	if err != nil {
		return nil, err
	}

	// Create ProcessedTimestamp (set to current time, will be updated in usecase)
	processedAt, err := domain.NewProcessedTimestamp(pm.clock.Now())
	if err != nil {
		return nil, err
	}

	// Create TraceID (already validated in middleware, but wrap here)
	traceIDObj, err := domain.NewTraceID(traceID)
	if err != nil {
		return nil, err
	}

	// Create IdempotencyKey if provided (may be nil)
	var idempotencyKeyObj *domain.IdempotencyKey
	if idempotencyKey != nil && *idempotencyKey != "" {
		obj, err := domain.NewIdempotencyKey(*idempotencyKey)
		if err != nil {
			return nil, err
		}
		idempotencyKeyObj = obj
	}

	// Generate EventID (UUID v4)
	eventIDStr := pm.idGenerator.Generate()
	eventID, err := domain.NewEventID(eventIDStr)
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
//
// Verifies: P2 (Async+202), P4 (4-Timestamps), P9 (Trace-ID)
func (pm *PositionMapper) FromEntity(entity *domain.VehiclePosition) *dto.PositionResponse {
	if entity == nil {
		return nil
	}

	return &dto.PositionResponse{
		EventID:   entity.EventID().String(),
		Status:    "accepted",
		Timestamp: pm.clock.Now(),
	}
}
