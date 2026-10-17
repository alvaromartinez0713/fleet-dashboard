package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/fleet-platform/backend/internal/ingestion/ports"
)

// EnvConfigLoader loads configuration from environment variables.
// It reads only the variables required for the ingestion service bootstrap:
// HTTP_PORT, HTTP_READ_TIMEOUT, HTTP_WRITE_TIMEOUT, HTTP_SHUTDOWN_TIMEOUT,
// LOG_LEVEL, LOG_FORMAT.
// Kafka and PostgreSQL configuration is not included in this loader and will
// be added in the ingestion spec.
type EnvConfigLoader struct{}

// NewEnvConfigLoader creates a new environment-based configuration loader.
func NewEnvConfigLoader() *EnvConfigLoader {
	return &EnvConfigLoader{}
}

// Load reads configuration from environment variables and returns a Config struct.
// If any required variable is missing or has an invalid format, it returns an error
// that identifies the field name without including the field value.
func (l *EnvConfigLoader) Load() (ports.Config, error) {
	var cfg ports.Config

	// Load HTTP_PORT with default value
	portStr := getEnvWithDefault("HTTP_PORT", "8080")
	port, err := parseInt(portStr, "HTTP_PORT")
	if err != nil {
		return cfg, err
	}
	cfg.HTTP.Port = port

	// Load HTTP_READ_TIMEOUT with default value
	readTimeoutStr := getEnvWithDefault("HTTP_READ_TIMEOUT", "15s")
	readTimeout, err := parseDuration(readTimeoutStr, "HTTP_READ_TIMEOUT")
	if err != nil {
		return cfg, err
	}
	cfg.HTTP.ReadTimeout = readTimeout

	// Load HTTP_WRITE_TIMEOUT with default value
	writeTimeoutStr := getEnvWithDefault("HTTP_WRITE_TIMEOUT", "15s")
	writeTimeout, err := parseDuration(writeTimeoutStr, "HTTP_WRITE_TIMEOUT")
	if err != nil {
		return cfg, err
	}
	cfg.HTTP.WriteTimeout = writeTimeout

	// Load HTTP_SHUTDOWN_TIMEOUT with default value
	shutdownTimeoutStr := getEnvWithDefault("HTTP_SHUTDOWN_TIMEOUT", "30s")
	shutdownTimeout, err := parseDuration(shutdownTimeoutStr, "HTTP_SHUTDOWN_TIMEOUT")
	if err != nil {
		return cfg, err
	}
	cfg.HTTP.ShutdownTimeout = shutdownTimeout

	// Load LOG_LEVEL with default value
	cfg.Logging.Level = getEnvWithDefault("LOG_LEVEL", "info")

	// Load LOG_FORMAT with default value
	cfg.Logging.Format = getEnvWithDefault("LOG_FORMAT", "json")

	return cfg, nil
}

// getEnvRequired retrieves the value of an environment variable.
// If the variable is not defined, it returns an error that identifies the field name.
func getEnvRequired(key string) (string, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return "", fmt.Errorf("required configuration field %s is not defined", key)
	}
	return value, nil
}

// getEnvWithDefault retrieves the value of an environment variable.
// If the variable is not defined, it returns the provided default value.
func getEnvWithDefault(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists || value == "" {
		return defaultValue
	}
	return value
}

// parseInt parses a string value as an integer.
// If the format is incorrect, it returns an error that identifies the field name
// without including the actual value.
func parseInt(value, field string) (int, error) {
	result, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("configuration field %s has invalid integer format", field)
	}
	return result, nil
}

// parseDuration parses a string value as a time.Duration.
// If the format is incorrect, it returns an error that identifies the field name
// without including the actual value.
func parseDuration(value, field string) (time.Duration, error) {
	result, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("configuration field %s has invalid duration format", field)
	}
	return result, nil
}
