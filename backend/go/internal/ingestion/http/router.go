package ingestionhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fleet-platform/backend/internal/ingestion/http/handler"
	"github.com/fleet-platform/backend/internal/ingestion/http/health"
	"github.com/fleet-platform/backend/internal/ingestion/http/mapper"
	"github.com/fleet-platform/backend/internal/ingestion/http/middleware"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// Router wraps the Gin engine and http.Server, implementing the ports.HTTPServer interface.
// It coordinates middleware registration, route registration, and server lifecycle.
type Router struct {
	engine      *gin.Engine
	server      *http.Server
	cfg         ports.Config
	logger      ports.Logger
	healthCheck ports.HealthChecker
}

// NewRouter creates a new Router with the provided configuration, logger, and health checker.
// It initializes the Gin engine with minimal setup.
// Middleware and routes are registered separately via RegisterRoutes() for explicit control.
func NewRouter(
	cfg ports.Config,
	logger ports.Logger,
	healthChecker ports.HealthChecker,
) *Router {
	// Create a new Gin engine without default middleware
	engine := gin.New()

	// Enable 405 Method Not Allowed responses
	engine.HandleMethodNotAllowed = true

	// Create router instance
	r := &Router{
		engine:      engine,
		cfg:         cfg,
		logger:      logger,
		healthCheck: healthChecker,
	}

	// Register custom handlers for NoRoute (404) and NoMethod (405)
	r.registerErrorHandlers()

	// Register health endpoint (static, always present)
	r.registerHealthRoute()

	// Create the http.Server with configured timeouts
	r.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      engine,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	return r
}

// RegisterRoutes configures the middleware pipeline in EXACT ORDER and registers routes.
// This must be called exactly once before Start().
//
// Middleware Execution Order (CRITICAL):
// 1. TraceIDMiddleware           (outermost, wraps all - first registered)
// 2. IdempotencyKeyMiddleware    (second)
// 3. ContentTypeMiddleware       (third)
// 4. RateLimitMiddleware         (fourth)
// 5. ErrorHandlerMiddleware      (innermost, last registered)
//
// NOTE: Gin executes middleware in REVERSE order of registration,
// so ErrorHandler is registered LAST but executes INNERMOST.
//
// Routes are registered AFTER all middleware.
//
// Verifies: All correctness properties, RF-01 to RF-13
func (r *Router) RegisterRoutes(
	positionHandler *handler.PositionHandler,
	errorMapper *mapper.ErrorMapper,
	rateLimiter ports.RateLimiter,
	clock ports.Clock,
) error {
	// ========== MIDDLEWARE PIPELINE (CRITICAL - exact order) ==========
	// Register in REVERSE execution order (last registered = innermost = executed last)

	// 5. ErrorHandlerMiddleware - LAST (will be innermost, executes AFTER all handlers)
	// This catches all errors and panics, mapping them to RFC 9457 ProblemDetails
	r.engine.Use(middleware.ErrorHandlerMiddleware(errorMapper))

	// 4. RateLimitMiddleware - FOURTH
	// Verifies rate limit per vehicle_id (12/60s)
	// Returns 429 Too Many Requests if exceeded
	r.engine.Use(middleware.RateLimitMiddleware(rateLimiter, clock))

	// 3. ContentTypeMiddleware - THIRD
	// Validates Content-Type: application/json for POST/PATCH/PUT
	// Returns 415 Unsupported Media Type if invalid
	r.engine.Use(middleware.ContentTypeMiddleware())

	// 2. IdempotencyKeyMiddleware - SECOND
	// Validates X-Idempotency-Key header (optional, if provided must be UUID v4)
	// Stores idempotency_key in context for downstream handlers
	r.engine.Use(middleware.IdempotencyKeyMiddleware())

	// 1. TraceIDMiddleware - FIRST (will be outermost, executes BEFORE all handlers)
	// Generates or validates X-Trace-ID header (UUID v4)
	// Stores trace_id in context for all downstream handlers
	// Always writes X-Trace-ID to response header
	r.engine.Use(middleware.TraceIDMiddleware())

	// ========== ROUTE REGISTRATION ==========
	// Routes are registered AFTER all middleware

	// POST /api/v1/positions - Position ingestion endpoint
	// Accepts: PositionRequest DTO
	// Returns: 202 Accepted (async processing) or 4xx/5xx with ProblemDetails
	// Verifies: P2 (Async+202), P4 (4-Timestamps), P9 (Trace-ID)
	r.engine.POST("/api/v1/positions", positionHandler.Handle)

	return nil
}

// registerHealthRoute registers the health check endpoint.
func (r *Router) registerHealthRoute() {
	healthHandler := health.NewHealthHandler(r.healthCheck)
	r.engine.GET("/health", healthHandler.Handle)
}

// registerErrorHandlers configures custom handlers for 404 (NoRoute) and 405 (NoMethod)
// responses with RFC 9457 Problem Details format.
func (r *Router) registerErrorHandlers() {
	r.engine.NoRoute(middleware.NoRouteHandler())
	r.engine.NoMethod(middleware.NoMethodHandler())
}

// Start binds the configured port and starts the HTTP server in a non-blocking goroutine.
// Returns an error if the port cannot be bound.
func (r *Router) Start() error {
	// Start the server in a goroutine so it doesn't block
	go func() {
		r.logger.Info("starting HTTP server",
			ports.Field{Key: "port", Value: r.cfg.HTTP.Port},
			ports.Field{Key: "read_timeout", Value: r.cfg.HTTP.ReadTimeout.String()},
			ports.Field{Key: "write_timeout", Value: r.cfg.HTTP.WriteTimeout.String()},
		)

		if err := r.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			r.logger.Fatal("HTTP server failed",
				ports.Field{Key: "error", Value: err.Error()},
			)
		}
	}()

	// Give the server a moment to fail if the port cannot be bound
	time.Sleep(100 * time.Millisecond)

	return nil
}

// Stop performs a graceful shutdown of the HTTP server using the provided context.
// It stops accepting new connections and waits for active requests to complete
// within the context deadline (typically cfg.HTTP.ShutdownTimeout).
// Returns an error if the shutdown process fails or times out.
func (r *Router) Stop(ctx context.Context) error {
	r.logger.Info("initiating graceful shutdown of HTTP server")

	if err := r.server.Shutdown(ctx); err != nil {
		r.logger.Error("HTTP server shutdown failed",
			ports.Field{Key: "error", Value: err.Error()},
		)
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	r.logger.Info("HTTP server shutdown completed successfully")
	return nil
}

// RegisterRoute adds a new route to the engine.
// Used by the composer to register routes after the router is created.
// Deprecated: Use RegisterRoutes() instead for better separation of concerns.
func (r *Router) RegisterRoute(method, path string, handlers ...gin.HandlerFunc) {
	r.engine.Handle(method, path, handlers...)
}

// GetEngine returns the underlying Gin engine.
// Used by the composer to register middleware and routes.
// This is useful for testing and direct access to the Gin engine if needed.
func (r *Router) GetEngine() *gin.Engine {
	return r.engine
}
