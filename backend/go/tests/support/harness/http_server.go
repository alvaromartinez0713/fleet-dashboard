package support

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/fleet-platform/backend/internal/ingestion/application/usecases"
	"github.com/fleet-platform/backend/internal/ingestion/http/handler"
	"github.com/fleet-platform/backend/internal/ingestion/http/mapper"
	"github.com/fleet-platform/backend/internal/ingestion/http/middleware"
	"github.com/fleet-platform/backend/internal/ingestion/ports"
	"github.com/fleet-platform/backend/tests/support/fakes"
)

// TestHTTPServer wraps the HTTP server with injected fakes for testing.
// It provides a complete test harness that:
// - Injects fake implementations for all external dependencies
// - Manages server lifecycle (start/stop)
// - Exposes a stable base URL for test requests
// - Provides access to fakes for verification
//
// This harness isolates the HTTP layer and handlers from actual dependencies,
// making tests fast, deterministic, and independent of infrastructure.
type TestHTTPServer struct {
	// Internal state
	server   *http.Server
	listener net.Listener
	baseURL  string

	// Fake implementations injected for testing
	clock        *fakes.FakeClock
	idGen        *fakes.FakeIDGenerator
	kafkaPub     *fakes.FakeKafkaPublisher
	rateLim      *fakes.FakeRateLimiter
	circuitBreak *fakes.FakeCircuitBreaker

	// Configuration for this test instance
	testConfig ports.Config
	testLogger ports.Logger
}

// NewTestHTTPServer creates a new TestHTTPServer with all dependencies injected as fakes.
// The server is created but not started; call Start() to begin accepting requests.
//
// Returns:
//   - *TestHTTPServer: Configured test server ready to start
//   - error: Any error during composition (e.g., invalid config)
//
// The returned server uses:
// - A fake clock (deterministic time)
// - A fake ID generator (predictable UUIDs)
// - A fake Kafka publisher (in-memory event storage)
// - A fake rate limiter (allows custom limits)
// - A fake circuit breaker (deterministic behavior)
func NewTestHTTPServer() (*TestHTTPServer, error) {
	// Create all fake implementations
	fakeClock := fakes.NewFakeClock()
	fakeIDGen := fakes.NewFakeIDGeneratorWithDefaults()
	fakeKafkaPub := &fakes.FakeKafkaPublisher{}
	fakeRateLim := fakes.NewFakeRateLimiter(12) // Default limit: 12 requests per window
	fakeCircuitBreak := &fakes.FakeCircuitBreaker{}

	// Create test configuration with reasonable defaults
	testCfg := ports.Config{}
	testCfg.HTTP.Port = 0 // Will use auto-assigned port
	testCfg.HTTP.ReadTimeout = 15 * time.Second
	testCfg.HTTP.WriteTimeout = 15 * time.Second
	testCfg.HTTP.ShutdownTimeout = 5 * time.Second
	testCfg.Logging.Level = "info"
	testCfg.Logging.Format = "json"

	// Create test logger
	testLogger := NewTestLogger()

	// Create test server instance
	tserver := &TestHTTPServer{
		clock:        fakeClock,
		idGen:        fakeIDGen,
		kafkaPub:     fakeKafkaPub,
		rateLim:      fakeRateLim,
		circuitBreak: fakeCircuitBreak,
		testConfig:   testCfg,
		testLogger:   testLogger,
	}

	// Compose the HTTP server with fakes
	if err := tserver.composeServer(); err != nil {
		return nil, fmt.Errorf("failed to compose test HTTP server: %w", err)
	}

	return tserver, nil
}

// composeServer wires together all components with injected fakes.
// This method:
// 1. Creates a Gin engine with fakes instead of real dependencies
// 2. Registers all middleware in the correct order
// 3. Sets up the HTTP server with a dynamically allocated port
// 4. Registers routes
func (s *TestHTTPServer) composeServer() error {
	// Step 1: Create Gin engine directly (Router.go not yet implemented)
	// TODO: Once router.go is implemented (task 8.4), use ingestionhttp.NewRouter()
	engine := gin.New()

	// Step 2: Create mappers with injected fakes
	errorMapper := mapper.NewErrorMapper()
	positionMapper := mapper.NewPositionMapper(s.idGen, s.clock)

	// Step 3: Create use case with fake dependencies
	// We use the real ReceivePositionUseCase from the application package
	// but with fake adapters for Kafka and circuit breaker
	receivePositionUseCase := usecases.NewReceivePositionUseCase(s.kafkaPub, s.circuitBreak)

	// Step 4: Create handler
	positionHandler := handler.NewPositionHandler(receivePositionUseCase, positionMapper)

	// Step 5: Register middleware in CORRECT ORDER
	// CRITICAL: ErrorHandlerMiddleware MUST be registered FIRST (outermost)
	// so it runs LAST (catches all errors from inner middleware and handlers)
	// Order of registration (innermost to outermost):
	//   1. ErrorHandlerMiddleware (registered first, runs last - catches all errors)
	//   2. RateLimitMiddleware
	//   3. ContentTypeMiddleware
	//   4. IdempotencyKeyMiddleware
	//   5. TraceIDMiddleware (registered last, runs first - generates trace ID)
	engine.Use(middleware.ErrorHandlerMiddleware(errorMapper))
	engine.Use(middleware.RateLimitMiddleware(s.rateLim, s.clock))
	engine.Use(middleware.ContentTypeMiddleware())
	engine.Use(middleware.IdempotencyKeyMiddleware())
	engine.Use(middleware.TraceIDMiddleware())

	// Step 6: Register routes
	engine.POST("/api/v1/positions", positionHandler.Handle)

	// Step 7: Create a listener on a random port (0 = auto-assign)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	s.listener = listener

	// Step 8: Extract the actual port and build base URL
	addr := listener.Addr().(*net.TCPAddr)
	s.baseURL = fmt.Sprintf("http://127.0.0.1:%d", addr.Port)

	// Step 9: Create the HTTP server with the listener
	s.server = &http.Server{
		Addr:         listener.Addr().String(),
		Handler:      engine,
		ReadTimeout:  s.testConfig.HTTP.ReadTimeout,
		WriteTimeout: s.testConfig.HTTP.WriteTimeout,
	}

	return nil
}

// Start begins accepting requests on the test server.
// The server listens on a random port (determined during composition).
// Returns an error if the server cannot start.
func (s *TestHTTPServer) Start() error {
	// Start the server in a goroutine
	go func() {
		if err := s.server.Serve(s.listener); err != nil && err != http.ErrServerClosed {
			// Log but don't panic; the test can handle connection errors
			s.testLogger.Error("test server listener error", ports.Field{Key: "error", Value: err.Error()})
		}
	}()

	// Give the server a moment to start
	time.Sleep(10 * time.Millisecond)

	return nil
}

// Stop gracefully shuts down the test server.
// Closes the listener and waits for active connections to complete.
// Returns an error if shutdown fails or times out.
func (s *TestHTTPServer) Stop() error {
	if s.server == nil {
		return nil
	}

	// Create a context with timeout for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), s.testConfig.HTTP.ShutdownTimeout)
	defer cancel()

	// Shutdown the server gracefully
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to stop test server: %w", err)
	}

	return nil
}

// BaseURL returns the base URL of the test server (e.g., "http://127.0.0.1:8080").
// Use this URL to construct requests in tests.
func (s *TestHTTPServer) BaseURL() string {
	return s.baseURL
}

// GetClock returns the fake clock for controlling time in tests.
// Use this to advance time or set specific timestamps.
func (s *TestHTTPServer) GetClock() *fakes.FakeClock {
	return s.clock
}

// GetIDGenerator returns the fake ID generator for controlling generated IDs.
// Use this to verify generated IDs or control the sequence.
func (s *TestHTTPServer) GetIDGenerator() *fakes.FakeIDGenerator {
	return s.idGen
}

// GetKafkaPublisher returns the fake Kafka publisher for verifying published events.
// Use this to check what events were published.
func (s *TestHTTPServer) GetKafkaPublisher() *fakes.FakeKafkaPublisher {
	return s.kafkaPub
}

// GetRateLimiter returns the fake rate limiter for testing rate limit behavior.
// Use this to verify rate limit enforcement.
func (s *TestHTTPServer) GetRateLimiter() *fakes.FakeRateLimiter {
	return s.rateLim
}

// GetCircuitBreaker returns the fake circuit breaker for testing resilience.
// Use this to verify circuit breaker behavior.
func (s *TestHTTPServer) GetCircuitBreaker() *fakes.FakeCircuitBreaker {
	return s.circuitBreak
}

// testHealthChecker is a minimal health checker for testing.
type testHealthChecker struct{}

// Check always returns healthy for tests.
func (t *testHealthChecker) Check(ctx context.Context) (bool, error) {
	return true, nil
}
