package dto

import "time"

// PositionRequest represents the incoming HTTP request body for position ingestion.
// Fields map directly to JSON and are validated by the HTTP adapter layer.
// This is a DTO (Data Transfer Object) separate from domain entities.
//
// Example JSON:
//   {
//     "vehicle_id": "VH-001",
//     "latitude": 40.7128,
//     "longitude": -74.0060,
//     "captured_at": "2024-01-15T10:30:00Z"
//   }
type PositionRequest struct {
	VehicleID   string    `json:"vehicle_id" binding:"required"`
	Latitude    float64   `json:"latitude" binding:"required"`
	Longitude   float64   `json:"longitude" binding:"required"`
	CapturedAt  time.Time `json:"captured_at" binding:"required"`
}
