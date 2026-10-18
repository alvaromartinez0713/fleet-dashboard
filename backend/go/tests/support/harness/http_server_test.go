package support

import (
	"testing"
)

// TestNewTestHTTPServer verifies that TestHTTPServer can be created and started.
func TestNewTestHTTPServer(t *testing.T) {
	server, err := NewTestHTTPServer()
	if err != nil {
		t.Fatalf("failed to create test server: %v", err)
	}

	if server == nil {
		t.Fatal("server is nil")
	}

	// Verify base URL is set
	if server.BaseURL() == "" {
		t.Fatal("base URL is empty")
	}

	// Verify getters work
	if server.GetClock() == nil {
		t.Fatal("clock is nil")
	}

	if server.GetIDGenerator() == nil {
		t.Fatal("id generator is nil")
	}

	if server.GetKafkaPublisher() == nil {
		t.Fatal("kafka publisher is nil")
	}

	if server.GetRateLimiter() == nil {
		t.Fatal("rate limiter is nil")
	}

	if server.GetCircuitBreaker() == nil {
		t.Fatal("circuit breaker is nil")
	}
}

// TestTestHTTPServerStart verifies that the server can start and stop cleanly.
func TestTestHTTPServerStart(t *testing.T) {
	server, err := NewTestHTTPServer()
	if err != nil {
		t.Fatalf("failed to create test server: %v", err)
	}

	err = server.Start()
	if err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	// Verify server is listening
	baseURL := server.BaseURL()
	if baseURL == "" {
		t.Fatal("base URL should not be empty after start")
	}

	// Stop the server
	err = server.Stop()
	if err != nil {
		t.Fatalf("failed to stop server: %v", err)
	}
}
