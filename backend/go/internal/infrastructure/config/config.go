package config

// Config holds all application configuration from environment variables.
type Config struct {
	Database DatabaseConfig
}

// DatabaseConfig holds database-specific configuration.
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// ValidationError represents a configuration validation error.
type ValidationError struct {
	Field   string
	Message string
}
