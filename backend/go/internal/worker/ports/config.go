package ports

import "time"

// Config holds the configuration for the worker service.
// Configuration is loaded exclusively from environment variables using the WORKER_* naming convention.
type Config struct {
	Worker struct {
		// ShutdownTimeout is the duration allowed for graceful shutdown before forcing termination.
		// env: WORKER_SHUTDOWN_TIMEOUT
		// default: 30s
		// range: 1-300s
		ShutdownTimeout time.Duration
	}
	Kafka struct {
		// Brokers is a slice of Kafka broker addresses (host:port).
		// env: WORKER_KAFKA_BROKERS (comma-separated, e.g., "localhost:9092,localhost:9093")
		// required: true
		Brokers []string

		// Topic is the Kafka topic from which the worker consumes position events.
		// env: WORKER_KAFKA_TOPIC
		// default: "positions"
		Topic string

		// ConsumerGroup is the Kafka consumer group to which this worker belongs.
		// env: WORKER_KAFKA_CONSUMER_GROUP
		// default: "telemetry-worker"
		ConsumerGroup string

		// BatchMaxSize is the maximum number of messages to batch before flushing offset commits.
		// env: WORKER_KAFKA_BATCH_MAX_SIZE
		// default: 500
		// range: 1-5000
		BatchMaxSize int
	}
	Retry struct {
		// MaxRetries is the maximum number of retry attempts for recoverable errors.
		// env: WORKER_RETRY_MAX_RETRIES
		// default: 3
		// range: 0-10
		MaxRetries int

		// BaseBackoff is the initial backoff duration before the first retry attempt.
		// env: WORKER_RETRY_BASE_BACKOFF
		// default: 100ms
		BaseBackoff time.Duration

		// BackoffFactor is the exponential multiplier applied to the backoff duration after each retry.
		// env: WORKER_RETRY_BACKOFF_FACTOR
		// default: 2.0
		// format: float64
		BackoffFactor float64
	}
	CircuitBreaker struct {
		// FailureThreshold is the number of consecutive recoverable errors that trigger the circuit breaker
		// to transition from CLOSED to OPEN state.
		// env: WORKER_CB_FAILURE_THRESHOLD
		// default: 5
		// range: >= 1
		FailureThreshold int

		// RecoveryInterval is the duration the circuit breaker remains in OPEN state before attempting
		// to transition to HALF_OPEN and test recovery.
		// env: WORKER_CB_RECOVERY_INTERVAL
		// default: 30s
		RecoveryInterval time.Duration
	}
	Database struct {
		// DSN is the PostgreSQL connection string (Data Source Name).
		// env: WORKER_DB_DSN
		// required: true
		// format: "postgres://user:password@localhost:5432/fleet_db?sslmode=disable"
		DSN string

		// OperationTimeout is the maximum duration for each database operation (query execution, commit).
		// env: WORKER_DB_OPERATION_TIMEOUT
		// default: 5s
		OperationTimeout time.Duration

		// MaxOpenConns is the maximum number of open connections to the database.
		// env: WORKER_DB_MAX_OPEN_CONNS
		// default: 10
		// range: >= 1
		MaxOpenConns int

		// MaxIdleConns is the maximum number of idle connections maintained in the connection pool.
		// env: WORKER_DB_MAX_IDLE_CONNS
		// default: 5
		// range: 0 to MaxOpenConns
		MaxIdleConns int
	}
	Logging struct {
		// Level is the logging level (debug, info, warn, error).
		// env: LOG_LEVEL
		// default: "info"
		Level string

		// Format is the logging format (json, text).
		// env: LOG_FORMAT
		// default: "json"
		Format string
	}
}

// ConfigLoader defines the interface for loading configuration from environment variables.
type ConfigLoader interface {
	Load() (Config, error)
}

// Validator defines the interface for validating configuration values.
type Validator interface {
	Validate(cfg Config) []ValidationError
}

// ValidationError represents a single configuration validation error.
// The error identifies the field and provides a message without including the field value.
type ValidationError struct {
	Field   string // Name of the configuration field that failed validation
	Rule    string // The validation rule that was violated
	Message string // Human-readable error message without the field value
}
