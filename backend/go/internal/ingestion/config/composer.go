package config

import (
	"context"

	"github.com/fleet-platform/backend/internal/ingestion/adapters/outbound/infrastructure"
	"github.com/fleet-platform/backend/internal/ingestion/adapters/outbound/messaging"
	"github.com/fleet-platform/backend/internal/ingestion/adapters/outbound/resilience"
	"github.com/fleet-platform/backend/internal/ingestion/application/usecases"
	"github.com/fleet-platform/backend/internal/ingestion/domain"
	ingestionhttp "github.com/fleet-platform/backend/internal/ingestion/http"
	"github.com/fleet-platform/backend/internal/ingestion/http/handler"
	"github.com/fleet-platform/backend/internal/ingestion/http/mapper"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// DependencyComposer wires together all dependencies for the ingestion service.
// It is responsible for instantiating and connecting adapters, handlers, and infrastructure
// components according to the provided configuration.
//
// Responsibility: Dependency Injection (DI) only.
// No business logic, no orchestration beyond wiring.
// Only composition of ports (interfaces) with adapters (implementations).
type DependencyComposer struct {
	cfg    ports.Config
	logger ports.Logger

	// Reusable infrastructure adapters
	clock       ports.Clock
	idGenerator ports.IDGenerator
}

// NewDependencyComposer creates a new DependencyComposer with the provided configuration and logger.
func NewDependencyComposer(cfg ports.Config, logger ports.Logger) *DependencyComposer {
	return &DependencyComposer{
		cfg:    cfg,
		logger: logger,
	}
}

// ComposeAPI wires together the HTTP server and all its dependencies.
// Implements Clean Architecture: instantiates adapters, ports, use cases, handlers, middleware,
// and registers routes in correct order.
//
// Middleware Pipeline Order (CRITICAL):
//   1. TraceIDMiddleware (first, wraps all)
//   2. IdempotencyKeyMiddleware (validates idempotency key)
//   3. ContentTypeMiddleware (validates Content-Type)
//   4. RateLimitMiddleware (enforces rate limit per vehicle)
//   5. ErrorHandlerMiddleware (last, catches all errors)
//
// Returns:
//   - ports.HTTPServer: The configured and ready-to-start HTTP server
//   - error: Any error encountered during composition
func (c *DependencyComposer) ComposeAPI() (ports.HTTPServer, error) {
	// ========== STEP 1: Create infrastructure adapters ==========
	c.clock = infrastructure.NewRealClock()
	c.idGenerator = infrastructure.NewRealIDGenerator()

	// ========== STEP 2: Create outbound adapters (messaging & resilience) ==========
	// Kafka Publisher
	brokers := []string{"localhost:9092"} // Will be parameterized via config
	kafkaPublisher, err := messaging.NewKafkaPublisher(brokers)
	if err != nil {
		c.logger.Error("failed to create Kafka publisher", ports.Field{Key: "error", Value: err.Error()})
		return nil, err
	}

	// Circuit Breaker (resilience)
	circuitBreaker := resilience.NewCircuitBreaker()

	// Rate Limiter (resilience)
	rateLimitConfig, err := domain.NewRateLimitConfig(12, 60)
	if err != nil {
		c.logger.Error("failed to create rate limit config", ports.Field{Key: "error", Value: err.Error()})
		return nil, err
	}
	rateLimiter := resilience.NewRateLimiter(rateLimitConfig)

	// ========== STEP 3: Create HTTP layer mappers ==========
	errorMapper := mapper.NewErrorMapper()
	positionMapper := mapper.NewPositionMapper(c.idGenerator, c.clock)

	// ========== STEP 4: Create application use cases ==========
	// ReceivePositionUseCase depends on ports (interfaces), not implementations
	receivePositionUseCase := usecases.NewReceivePositionUseCase(kafkaPublisher, circuitBreaker)

	// ========== STEP 5: Create HTTP layer handlers ==========
	positionHandler := handler.NewPositionHandler(receivePositionUseCase, positionMapper)

	// ========== STEP 6: Create health checker ==========
	healthChecker := NewSimpleHealthChecker()

	// ========== STEP 7: Create base router (framework setup only) ==========
	router := ingestionhttp.NewRouter(c.cfg, c.logger, healthChecker)

	// ========== STEP 8: Register middleware in CORRECT ORDER ==========
	// Router.RegisterRoutes() handles middleware pipeline configuration and route registration
	// with strict ordering guarantees (middleware executed in reverse order of registration)
	if err := router.RegisterRoutes(positionHandler, errorMapper, rateLimiter, c.clock); err != nil {
		c.logger.Error("failed to register routes", ports.Field{Key: "error", Value: err.Error()})
		return nil, err
	}

	c.logger.Info("API dependencies composed successfully",
		ports.Field{Key: "middleware_count", Value: 5},
		ports.Field{Key: "middleware_order", Value: "TraceID → Idempotency → ContentType → RateLimit → ErrorHandler"},
		ports.Field{Key: "routes_registered", Value: 1},
		ports.Field{Key: "route_position_ingestion", Value: "POST /api/v1/positions"},
	)

	return router, nil
}

// SimpleHealthChecker is a basic implementation of ports.HealthChecker
type SimpleHealthChecker struct{}

// NewSimpleHealthChecker creates a new SimpleHealthChecker.
func NewSimpleHealthChecker() *SimpleHealthChecker {
	return &SimpleHealthChecker{}
}

// Check always returns (true, nil) indicating the service is healthy.
func (s *SimpleHealthChecker) Check(ctx context.Context) (bool, error) {
	return true, nil
}
