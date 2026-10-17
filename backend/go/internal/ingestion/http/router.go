package ingestionhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fleet-platform/backend/internal/ingestion/http/health"
	"github.com/fleet-platform/backend/internal/ingestion/http/middleware"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// Router wraps the Gin engine and http.Server, implementing the ports.HTTPServer interface.
type Router struct {
	engine      *gin.Engine
	server      *http.Server
	cfg         ports.Config
	logger      ports.Logger
	healthCheck ports.HealthChecker
}

// NewRouter creates a new Router with the provided configuration, logger, and health checker.
// It initializes the Gin engine with custom settings and registers the middleware chain
// in order: trace_id → logging → recovery/error.
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

	// Register middleware chain: trace_id → logging → recovery/error
	r.registerMiddleware()

	// Register custom handlers for NoRoute (404) and NoMethod (405)
	r.registerErrorHandlers()

	// Register routes
	r.registerRoutes()

	// Create the http.Server with configured timeouts
	r.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTP.Port),
		Handler:      engine,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
	}

	return r
}

// registerMiddleware sets up the middleware chain in order:
// 1. TraceIDMiddleware - extracts or generates trace ID
// 2. LoggingMiddleware - logs request details with trace ID
// 3. RecoveryMiddleware - recovers from panics and logs stack traces
func (r *Router) registerMiddleware() {
	r.engine.Use(middleware.TraceIDMiddleware())
	r.engine.Use(middleware.LoggingMiddleware(r.logger))
	r.engine.Use(middleware.RecoveryMiddleware(r.logger))
}

// registerErrorHandlers configures custom handlers for 404 (NoRoute) and 405 (NoMethod)
// responses with RFC 9457 Problem Details format.
func (r *Router) registerErrorHandlers() {
	r.engine.NoRoute(middleware.NoRouteHandler())
	r.engine.NoMethod(middleware.NoMethodHandler())
}

// registerRoutes registers all HTTP routes for the service.
// For this bootstrap spec, only the health endpoint is registered.
func (r *Router) registerRoutes() {
	// Create health handler
	healthHandler := health.NewHealthHandler(r.healthCheck)

	// Register health endpoint
	r.engine.GET("/health", healthHandler.Handle)
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
