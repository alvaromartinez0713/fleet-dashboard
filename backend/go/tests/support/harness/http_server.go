package harness

import (
	"context"
	"fmt"
	"io"
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
	httpConfig HTTPConfig
	testLogger ports.Logger
}

// HTTPConfig holds HTTP server configuration for testing
type HTTPConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration
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
	httpCfg := HTTPConfig{
		Port:            0, // Will use auto-assigned port
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		ShutdownTimeout: 5 * time.Second,
	}

	// Create test logger
	testLogger := NewTestLogger()

	// Create test server instance
	tserver := &TestHTTPServer{
		clock:        fakeClock,
		idGen:        fakeIDGen,
		kafkaPub:     fakeKafkaPub,
		rateLim:      fakeRateLim,
		circuitBreak: fakeCircuitBreak,
		httpConfig:   httpCfg,
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
		ReadTimeout:  s.httpConfig.ReadTimeout,
		WriteTimeout: s.httpConfig.WriteTimeout,
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

	// Create graceful shutdown context
	ctx, cancel := context.WithTimeout(context.Background(), s.httpConfig.ShutdownTimeout)
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


// NewTestHTTPServerWithPublisher creates a test HTTP server with a provided Kafka publisher.
// This is used for integration tests where a real Kafka publisher is needed.
//
// Args:
//   kafkaPublisher: A KafkaPublisher implementation (can be real or fake)
//
// Returns:
//   *TestHTTPServer: Configured test server ready to start
//   error: Any error during composition
//
// This variant allows tests to inject a real KafkaPublisher (connected to real Kafka)
// while keeping other dependencies as fakes for determinism.
func NewTestHTTPServerWithPublisher(kafkaPublisher ports.KafkaPublisher) *TestHTTPServer {
	// Create fake implementations for everything except Kafka
	fakeClock := fakes.NewFakeClock()
	fakeIDGen := fakes.NewFakeIDGeneratorWithDefaults()
	fakeRateLim := fakes.NewFakeRateLimiter(12)
	fakeCircuitBreak := &fakes.FakeCircuitBreaker{}

	// Create test configuration
	httpCfg := HTTPConfig{
		Port:            0,
		ReadTimeout:     15 * time.Second,
		WriteTimeout:    15 * time.Second,
		ShutdownTimeout: 5 * time.Second,
	}

	testLogger := NewTestLogger()

	// Create server with provided Kafka publisher
	tserver := &TestHTTPServer{
		clock:        fakeClock,
		idGen:        fakeIDGen,
		kafkaPub:     &fakes.FakeKafkaPublisher{}, // Placeholder, replaced below
		rateLim:      fakeRateLim,
		circuitBreak: fakeCircuitBreak,
		httpConfig:   httpCfg,
		testLogger:   testLogger,
	}

	// Compose the server (similar to composeServer but with real Kafka)
	engine := gin.New()
	errorMapper := mapper.NewErrorMapper()
	positionMapper := mapper.NewPositionMapper(tserver.idGen, tserver.clock)

	// Create use case with provided Kafka publisher
	receivePositionUseCase := usecases.NewReceivePositionUseCase(kafkaPublisher, tserver.circuitBreak)

	positionHandler := handler.NewPositionHandler(receivePositionUseCase, positionMapper)

	// Register middleware
	engine.Use(middleware.ErrorHandlerMiddleware(errorMapper))
	engine.Use(middleware.RateLimitMiddleware(tserver.rateLim, tserver.clock))
	engine.Use(middleware.ContentTypeMiddleware())
	engine.Use(middleware.IdempotencyKeyMiddleware())
	engine.Use(middleware.TraceIDMiddleware())

	// Register routes
	engine.POST("/api/v1/positions", positionHandler.Handle)

	// Create listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		tserver.testLogger.Error("failed to create listener", ports.Field{Key: "error", Value: err.Error()})
		return tserver
	}
	tserver.listener = listener

	// Extract port and build base URL
	addr := listener.Addr().(*net.TCPAddr)
	tserver.baseURL = fmt.Sprintf("http://127.0.0.1:%d", addr.Port)

	// Create HTTP server
	tserver.server = &http.Server{
		Addr:         listener.Addr().String(),
		Handler:      engine,
		ReadTimeout:  tserver.httpConfig.ReadTimeout,
		WriteTimeout: tserver.httpConfig.WriteTimeout,
	}

	// Start the server
	go func() {
		if err := tserver.server.Serve(tserver.listener); err != nil && err != http.ErrServerClosed {
			tserver.testLogger.Error("test server listener error", ports.Field{Key: "error", Value: err.Error()})
		}
	}()

	// Give server time to start
	time.Sleep(10 * time.Millisecond)

	return tserver
}

// BytesReader wraps a byte slice as an io.Reader
type BytesReader struct {
	data []byte
	pos  int
}

// NewBytesReader creates a new BytesReader from byte slice
func NewBytesReader(data []byte) *BytesReader {
	return &BytesReader{
		data: data,
		pos:  0,
	}
}

// Read implements io.Reader interface
func (br *BytesReader) Read(p []byte) (int, error) {
	if br.pos >= len(br.data) {
		return 0, io.EOF
	}
	n := copy(p, br.data[br.pos:])
	br.pos += n
	return n, nil
}
