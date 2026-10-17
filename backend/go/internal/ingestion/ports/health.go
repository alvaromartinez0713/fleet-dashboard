package ports

import "context"

// HealthChecker defines the contract for checking the operational status of the service.
type HealthChecker interface {
	// Check verifies whether the service and its required dependencies are operational.
	// Returns (true, nil) if the service is healthy and ready to serve requests.
	// Returns (false, error) if the service or a required dependency is unavailable.
	// The error provides diagnostic information but should never be exposed to external clients.
	Check(ctx context.Context) (bool, error)
}
