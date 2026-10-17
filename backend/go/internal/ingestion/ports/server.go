package ports

import "context"

// HTTPServer defines the contract for starting and stopping the HTTP server.
type HTTPServer interface {
	// Start binds the configured port and starts accepting HTTP requests.
	// Returns an error if the port cannot be bound or the server fails to start.
	Start() error

	// Stop performs a graceful shutdown of the HTTP server.
	// It stops accepting new connections and waits for active requests to complete
	// within the deadline provided by the context.
	// Returns an error if the shutdown process fails or times out.
	Stop(ctx context.Context) error
}
