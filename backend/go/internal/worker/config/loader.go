package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/fleet-platform/backend/internal/worker/ports"
)

// EnvConfigLoader loads configuration from environment variables.
// It reads only WORKER_SHUTDOWN_TIMEOUT, LOG_LEVEL, and LOG_FORMAT.
// All fields are loaded exclusively from environment variables with no fallback to files or other sources.
type EnvConfigLoader struct{}

// NewEnvConfigLoader creates a new environment-based configuration loader.
func NewEnvConfigLoader() *EnvConfigLoader {
	return &EnvConfigLoader{}
}

// Load reads configuration from environment variables and returns a Config struct.
// If any required field is missing, has incorrect type, or is out of range,
// it returns an error that identifies the field name without including its value.
func (l *EnvConfigLoader) Load() (ports.Config, error) {
	var cfg ports.Config

	// Load WORKER_SHUTDOWN_TIMEOUT (optional, default: 30s)
	shutdownTimeoutStr := os.Getenv("WORKER_SHUTDOWN_TIMEOUT")
	if shutdownTimeoutStr == "" {
		cfg.Worker.ShutdownTimeout = 30 * time.Second
	} else {
		shutdownTimeout, err := parseDuration(shutdownTimeoutStr, "WORKER_SHUTDOWN_TIMEOUT")
		if err != nil {
			return cfg, err
		}
		cfg.Worker.ShutdownTimeout = shutdownTimeout
	}

	// Load LOG_LEVEL (optional, default: "info")
	cfg.Logging.Level = os.Getenv("LOG_LEVEL")
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}

	// Load LOG_FORMAT (optional, default: "json")
	cfg.Logging.Format = os.Getenv("LOG_FORMAT")
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}

	return cfg, nil
}

// getEnvRequired retrieves a required environment variable.
// Returns an error identifying the field if the variable is not defined.
func getEnvRequired(key string) (string, error) {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return value, nil
}

// parseInt parses a string value as an integer.
// Returns an error identifying the field if the format is incorrect.
func parseInt(value, field string) (int, error) {
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("field %s has invalid integer format", field)
	}
	return result, nil
}

// parseDuration parses a string value as a time.Duration.
// Returns an error identifying the field if the format is incorrect.
func parseDuration(value, field string) (time.Duration, error) {
	result, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("field %s has invalid duration format", field)
	}
	return result, nil
}
