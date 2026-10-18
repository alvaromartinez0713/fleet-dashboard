package fakes

import (
	"context"
	"errors"
	"testing"

	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// TestFakeCircuitBreakerClosed verifies that Execute calls the function when circuit is closed.
func TestFakeCircuitBreakerClosed(t *testing.T) {
	breaker := &FakeCircuitBreaker{}

	called := false
	err := breaker.Execute(context.Background(), func(ctx context.Context) error {
		called = true
		return nil
	})

	if !called {
		t.Fatal("function was not called when circuit was closed")
	}

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if breaker.GetExecutionCount() != 1 {
		t.Fatalf("expected execution count 1, got %d", breaker.GetExecutionCount())
	}
}

// TestFakeCircuitBreakerOpen verifies that Execute returns error when circuit is open.
func TestFakeCircuitBreakerOpen(t *testing.T) {
	breaker := &FakeCircuitBreaker{}
	breaker.SetOpen()

	called := false
	err := breaker.Execute(context.Background(), func(ctx context.Context) error {
		called = true
		return nil
	})

	if called {
		t.Fatal("function was called when circuit was open")
	}

	if err == nil {
		t.Fatal("expected error when circuit is open")
	}

	_, ok := err.(ports.CircuitBreakerOpenError)
	if !ok {
		t.Fatalf("expected CircuitBreakerOpenError, got %T", err)
	}

	if breaker.GetExecutionCount() != 1 {
		t.Fatalf("expected execution count 1, got %d", breaker.GetExecutionCount())
	}
}

// TestFakeCircuitBreakerCallResults verifies that execution results are tracked.
func TestFakeCircuitBreakerCallResults(t *testing.T) {
	breaker := &FakeCircuitBreaker{}

	// First call - success
	breaker.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	// Second call - error
	expectedErr := errors.New("test error")
	breaker.Execute(context.Background(), func(ctx context.Context) error {
		return expectedErr
	})

	// Third call - circuit open
	breaker.SetOpen()
	breaker.Execute(context.Background(), func(ctx context.Context) error {
		return nil
	})

	results := breaker.GetCallResults()
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	// Verify first result
	if results[0].Error != nil {
		t.Fatalf("expected no error for first call, got %v", results[0].Error)
	}
	if !results[0].FunctionExecuted {
		t.Fatal("function should have executed for first call")
	}

	// Verify second result
	if results[1].Error != expectedErr {
		t.Fatalf("expected error %v for second call, got %v", expectedErr, results[1].Error)
	}
	if !results[1].FunctionExecuted {
		t.Fatal("function should have executed for second call")
	}

	// Verify third result
	if results[2].Error == nil {
		t.Fatal("expected error for third call (circuit open)")
	}
	if results[2].FunctionExecuted {
		t.Fatal("function should not have executed for third call (circuit open)")
	}
}

// TestFakeCircuitBreakerReset verifies that Reset clears state.
func TestFakeCircuitBreakerReset(t *testing.T) {
	breaker := &FakeCircuitBreaker{}
	breaker.SetOpen()
	breaker.Execute(context.Background(), func(ctx context.Context) error { return nil })

	breaker.Reset()

	if breaker.IsOpen() {
		t.Fatal("circuit should be closed after reset")
	}

	if breaker.GetExecutionCount() != 0 {
		t.Fatalf("expected execution count 0 after reset, got %d", breaker.GetExecutionCount())
	}

	if len(breaker.GetCallResults()) != 0 {
		t.Fatalf("expected 0 call results after reset, got %d", len(breaker.GetCallResults()))
	}
}
