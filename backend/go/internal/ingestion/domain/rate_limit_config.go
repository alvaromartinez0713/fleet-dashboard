package domain

import (
	"fmt"
	"time"
)

// RateLimitConfig is a Value Object representing rate limiting configuration.
// It is immutable after creation and validates invariants in the constructor.
//
// Validation rules:
// - requestsPerWindow must be > 0
// - windowSize must be > 0
//
// Default values:
// - requestsPerWindow: 12 requests
// - windowSize: 60 seconds
type RateLimitConfig struct {
	requestsPerWindow int
	windowSize        time.Duration
}

// NewRateLimitConfig creates a new RateLimitConfig with validation.
// Returns error if validation fails.
//
// Parameters:
// - requestsPerWindow: number of requests allowed per window (must be > 0)
// - windowSize: time duration of the window (must be > 0)
func NewRateLimitConfig(requestsPerWindow int, windowSize time.Duration) (*RateLimitConfig, error) {
	// Validate requestsPerWindow > 0
	if requestsPerWindow <= 0 {
		return nil, fmt.Errorf("requestsPerWindow must be greater than 0, got %d", requestsPerWindow)
	}

	// Validate windowSize > 0
	if windowSize <= 0 {
		return nil, fmt.Errorf("windowSize must be greater than 0, got %v", windowSize)
	}

	return &RateLimitConfig{
		requestsPerWindow: requestsPerWindow,
		windowSize:        windowSize,
	}, nil
}

// NewRateLimitConfigWithDefaults creates a new RateLimitConfig with default values.
// Default: 12 requests per 60 seconds.
func NewRateLimitConfigWithDefaults() *RateLimitConfig {
	config, _ := NewRateLimitConfig(12, 60*time.Second)
	return config
}

// RequestsPerWindow returns the number of requests allowed per window.
func (rc *RateLimitConfig) RequestsPerWindow() int {
	return rc.requestsPerWindow
}

// WindowSize returns the time duration of the rate limiting window.
func (rc *RateLimitConfig) WindowSize() time.Duration {
	return rc.windowSize
}

// String returns a string representation of the rate limit configuration.
func (rc *RateLimitConfig) String() string {
	return fmt.Sprintf("RateLimitConfig{requestsPerWindow: %d, windowSize: %v}", rc.requestsPerWindow, rc.windowSize)
}

// Equals checks if two RateLimitConfig values are equal.
func (rc *RateLimitConfig) Equals(other *RateLimitConfig) bool {
	if rc == nil || other == nil {
		return rc == other
	}
	return rc.requestsPerWindow == other.requestsPerWindow && rc.windowSize == other.windowSize
}
