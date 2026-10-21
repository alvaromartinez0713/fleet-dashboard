package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
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

	// Load WORKER_KAFKA_BROKERS (required, comma-separated)
	brokersStr := os.Getenv("WORKER_KAFKA_BROKERS")
	if brokersStr != "" {
		cfg.Kafka.Brokers = parseCommaSeparatedValues(brokersStr)
	}

	// Load WORKER_KAFKA_TOPIC (optional, default: "positions")
	cfg.Kafka.Topic = os.Getenv("WORKER_KAFKA_TOPIC")
	if cfg.Kafka.Topic == "" {
		cfg.Kafka.Topic = "positions"
	}

	// Load WORKER_KAFKA_CONSUMER_GROUP (optional, default: "telemetry-worker")
	cfg.Kafka.ConsumerGroup = os.Getenv("WORKER_KAFKA_CONSUMER_GROUP")
	if cfg.Kafka.ConsumerGroup == "" {
		cfg.Kafka.ConsumerGroup = "telemetry-worker"
	}

	// Load WORKER_KAFKA_BATCH_MAX_SIZE (optional, default: 500)
	batchMaxSizeStr := os.Getenv("WORKER_KAFKA_BATCH_MAX_SIZE")
	if batchMaxSizeStr == "" {
		cfg.Kafka.BatchMaxSize = 500
	} else {
		batchMaxSize, err := parseInt(batchMaxSizeStr, "WORKER_KAFKA_BATCH_MAX_SIZE")
		if err != nil {
			return cfg, err
		}
		cfg.Kafka.BatchMaxSize = batchMaxSize
	}

	// Load WORKER_RETRY_MAX_RETRIES (optional, default: 3)
	maxRetriesStr := os.Getenv("WORKER_RETRY_MAX_RETRIES")
	if maxRetriesStr == "" {
		cfg.Retry.MaxRetries = 3
	} else {
		maxRetries, err := parseInt(maxRetriesStr, "WORKER_RETRY_MAX_RETRIES")
		if err != nil {
			return cfg, err
		}
		cfg.Retry.MaxRetries = maxRetries
	}

	// Load WORKER_RETRY_BASE_BACKOFF (optional, default: 100ms)
	baseBackoffStr := os.Getenv("WORKER_RETRY_BASE_BACKOFF")
	if baseBackoffStr == "" {
		cfg.Retry.BaseBackoff = 100 * time.Millisecond
	} else {
		baseBackoff, err := parseDuration(baseBackoffStr, "WORKER_RETRY_BASE_BACKOFF")
		if err != nil {
			return cfg, err
		}
		cfg.Retry.BaseBackoff = baseBackoff
	}

	// Load WORKER_RETRY_BACKOFF_FACTOR (optional, default: 2.0)
	backoffFactorStr := os.Getenv("WORKER_RETRY_BACKOFF_FACTOR")
	if backoffFactorStr == "" {
		cfg.Retry.BackoffFactor = 2.0
	} else {
		backoffFactor, err := parseFloat64(backoffFactorStr, "WORKER_RETRY_BACKOFF_FACTOR")
		if err != nil {
			return cfg, err
		}
		cfg.Retry.BackoffFactor = backoffFactor
	}

	// Load WORKER_CB_FAILURE_THRESHOLD (optional, default: 5)
	failureThresholdStr := os.Getenv("WORKER_CB_FAILURE_THRESHOLD")
	if failureThresholdStr == "" {
		cfg.CircuitBreaker.FailureThreshold = 5
	} else {
		failureThreshold, err := parseInt(failureThresholdStr, "WORKER_CB_FAILURE_THRESHOLD")
		if err != nil {
			return cfg, err
		}
		cfg.CircuitBreaker.FailureThreshold = failureThreshold
	}

	// Load WORKER_CB_RECOVERY_INTERVAL (optional, default: 30s)
	recoveryIntervalStr := os.Getenv("WORKER_CB_RECOVERY_INTERVAL")
	if recoveryIntervalStr == "" {
		cfg.CircuitBreaker.RecoveryInterval = 30 * time.Second
	} else {
		recoveryInterval, err := parseDuration(recoveryIntervalStr, "WORKER_CB_RECOVERY_INTERVAL")
		if err != nil {
			return cfg, err
		}
		cfg.CircuitBreaker.RecoveryInterval = recoveryInterval
	}

	// Load WORKER_DB_DSN (required)
	cfg.Database.DSN = os.Getenv("WORKER_DB_DSN")

	// Load WORKER_DB_OPERATION_TIMEOUT (optional, default: 5s)
	operationTimeoutStr := os.Getenv("WORKER_DB_OPERATION_TIMEOUT")
	if operationTimeoutStr == "" {
		cfg.Database.OperationTimeout = 5 * time.Second
	} else {
		operationTimeout, err := parseDuration(operationTimeoutStr, "WORKER_DB_OPERATION_TIMEOUT")
		if err != nil {
			return cfg, err
		}
		cfg.Database.OperationTimeout = operationTimeout
	}

	// Load WORKER_DB_MAX_OPEN_CONNS (optional, default: 10)
	maxOpenConnsStr := os.Getenv("WORKER_DB_MAX_OPEN_CONNS")
	if maxOpenConnsStr == "" {
		cfg.Database.MaxOpenConns = 10
	} else {
		maxOpenConns, err := parseInt(maxOpenConnsStr, "WORKER_DB_MAX_OPEN_CONNS")
		if err != nil {
			return cfg, err
		}
		cfg.Database.MaxOpenConns = maxOpenConns
	}

	// Load WORKER_DB_MAX_IDLE_CONNS (optional, default: 5)
	maxIdleConnsStr := os.Getenv("WORKER_DB_MAX_IDLE_CONNS")
	if maxIdleConnsStr == "" {
		cfg.Database.MaxIdleConns = 5
	} else {
		maxIdleConns, err := parseInt(maxIdleConnsStr, "WORKER_DB_MAX_IDLE_CONNS")
		if err != nil {
			return cfg, err
		}
		cfg.Database.MaxIdleConns = maxIdleConns
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

// parseFloat64 parses a string value as a float64.
// Returns an error identifying the field if the format is incorrect.
func parseFloat64(value, field string) (float64, error) {
	result, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("field %s has invalid float64 format", field)
	}
	return result, nil
}

// parseCommaSeparatedValues splits a comma-separated string and returns a trimmed slice.
// Each value is trimmed of leading and trailing whitespace.
func parseCommaSeparatedValues(value string) []string {
	if value == "" {
		return []string{}
	}

	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
