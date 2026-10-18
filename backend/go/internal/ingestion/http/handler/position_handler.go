package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/fleet-platform/backend/internal/ingestion/application/usecases"
	"github.com/fleet-platform/backend/internal/ingestion/http/dto"
	"github.com/fleet-platform/backend/internal/ingestion/http/mapper"
	"github.com/fleet-platform/backend/internal/ingestion/http/middleware"
)

// PositionHandler handles HTTP POST /positions requests.
// Thin handler (~15 lines) following Clean Architecture:
// - Deserializes DTO
// - Extracts context (headers set by middleware)
// - Maps DTO to domain command
// - Executes use case
// - Maps entity to response DTO
// - Returns 202 Accepted
//
// Verifies: P2 (Async+202), RF-01, RF-04
type PositionHandler struct {
	useCase        *usecases.ReceivePositionUseCase
	positionMapper *mapper.PositionMapper
}

// NewPositionHandler creates a new position handler.
func NewPositionHandler(
	useCase *usecases.ReceivePositionUseCase,
	positionMapper *mapper.PositionMapper,
) *PositionHandler {
	return &PositionHandler{
		useCase:        useCase,
		positionMapper: positionMapper,
	}
}

// Handle processes POST /positions requests.
// Returns 202 Accepted if validation and async enqueuing succeed.
// Returns 4xx/5xx with ProblemDetails if validation or internal errors occur.
//
// Thin handler: ~15 lines of orchestration logic.
func (h *PositionHandler) Handle(c *gin.Context) {
	// 1. Deserialize DTO from JSON body
	var req dto.PositionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	// 2. Extract headers from middleware context
	traceID := middleware.GetTraceIDFromContext(c)
	idempotencyKey := middleware.GetIdempotencyKeyFromContext(c)

	// 3. Map DTO → Domain Command (validates inputs)
	command, err := h.positionMapper.FromRequest(&req, traceID, idempotencyKey)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// 4. Execute use case (creates entity, publishes async)
	entity, err := h.useCase.Execute(c.Request.Context(), command)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// 5. Map Entity → Response DTO
	response := h.positionMapper.FromEntity(entity)

	// 6. Return 202 Accepted (async enqueued)
	c.JSON(http.StatusAccepted, response)
}
