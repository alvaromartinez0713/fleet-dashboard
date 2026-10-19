package config

import (
	"fmt"
	"os"
)

// EnvConfigLoader loads configuration from environment variables.
type EnvConfigLoader struct{}

// NewEnvConfigLoader creates a new environment-based configuration loader.
func NewEnvConfigLoader() *EnvConfigLoader {
	return &EnvConfigLoader{}
}

// Load reads configuration from environment variables and returns a Config struct.
// Returns an error if any required variable is missing.
func (l *EnvConfigLoader) Load() (Config, error) {
	var cfg Config

	// Load database configuration from environment variables
	cfg.Database.Host = os.Getenv("DB_HOST")
	cfg.Database.Port = os.Getenv("DB_PORT")
	cfg.Database.User = os.Getenv("DB_USER")
	cfg.Database.Password = os.Getenv("DB_PASSWORD")
	cfg.Database.Name = os.Getenv("DB_NAME")

	return cfg, nil
}

// getEnvRequired retrieves the value of an environment variable.
// If the variable is not defined, it returns an error.
func getEnvRequired(key string) (string, error) {
	value, exists := os.LookupEnv(key)
	if !exists {
		return "", fmt.Errorf("required environment variable %s is not defined", key)
	}
	return value, nil
}
