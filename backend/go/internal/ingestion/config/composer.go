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
	"github.com/fleet-platform/backend/internal/ingestion/http/middleware"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// DependencyComposer wires together all dependencies for the ingestion service.
// It is responsible for instantiating and connecting adapters, handlers, and infrastructure
// components according to the provided configuration.
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
// It creates adapters, use cases, handlers, and the router with middleware.
//
// Returns:
//   - ports.HTTPServer: The configured and ready-to-start HTTP server
//   - error: Any error encountered during composition
func (c *DependencyComposer) ComposeAPI() (ports.HTTPServer, error) {
	// Create the health checker
	healthChecker := NewSimpleHealthChecker()

	// Create the HTTP router with all dependencies
	router := ingestionhttp.NewRouter(c.cfg, c.logger, healthChecker)

	// Create infrastructure adapters (injected for dependency management)
	c.clock = infrastructure.NewRealClock()
	c.idGenerator = infrastructure.NewRealIDGenerator()

	// Create Kafka publisher adapter
	brokers := []string{"localhost:9092"} // Will be parameterized via config
	kafkaPublisher, err := messaging.NewKafkaPublisher(brokers)
	if err != nil {
		c.logger.Error("failed to create Kafka publisher", ports.Field{Key: "error", Value: err.Error()})
		return nil, err
	}

	// Create circuit breaker manager
	circuitBreakerManager := resilience.NewCircuitBreakerManager()

	// Create rate limiter with default config (12/60s)
	rateLimitConfig, err := domain.NewRateLimitConfig(12, 60)
	if err != nil {
		c.logger.Error("failed to create rate limit config", ports.Field{Key: "error", Value: err.Error()})
		return nil, err
	}
	rateLimiter := resilience.NewRateLimiter(rateLimitConfig)

	// Create error mapper
	errorMapper := mapper.NewErrorMapper()

	// Create position mapper
	positionMapper := mapper.NewPositionMapper(c.idGenerator, c.clock)

	// Create use case
	receivePositionUseCase := usecases.NewReceivePositionUseCase(kafkaPublisher, circuitBreakerManager)

	// Create position handler
	positionHandler := handler.NewPositionHandler(receivePositionUseCase, positionMapper)

	// Silence unused variables (will be used in future waves)
	_ = circuitBreakerManager
	_ = rateLimiter
	_ = errorMapper
	_ = positionHandler

	// Register middleware and routes
	// Note: Middleware registration is handled in router.registerMiddleware()
	// Position handler registration will be done in a separate step

	c.logger.Info("API dependencies composed successfully")

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
