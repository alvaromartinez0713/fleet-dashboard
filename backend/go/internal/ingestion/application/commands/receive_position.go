package commands

import (
	"github.com/fleet-platform/backend/internal/ingestion/domain"
)

// ReceivePositionCommand represents the input to the ReceivePositionUseCase.
// It's a Data Transfer Object (DTO) internal to the application layer.
// Contains validated domain value objects (no raw strings/numbers).
//
// This command is created by the HTTP mapper from the request DTO
// and consumed by the UseCase to create the domain entity.
type ReceivePositionCommand struct {
	VehicleID      *domain.VehicleID
	EventID        domain.EventID
	GeoPosition    *domain.GeoPosition
	CapturedAt     domain.CapturedTimestamp
	ReceivedAt     domain.ReceivedTimestamp
	ProcessedAt    domain.ProcessedTimestamp
	TraceID        domain.TraceID
	IdempotencyKey *domain.IdempotencyKey // optional
}
