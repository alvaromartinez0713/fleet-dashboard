package config

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"

	"github.com/fleet-platform/backend/internal/worker/adapters/outbound/persistence"
	"github.com/fleet-platform/backend/internal/worker/ports"
)

// NewPostgresqlConnection creates a PostgreSQL/TimescaleDB connection pool.
// Returns: ports.PositionRepository (public port)
func NewPostgresqlConnection(dsn string, maxOpen, maxIdle int, operationTimeout time.Duration) (ports.PositionRepository, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(maxOpen)
	db.SetMaxIdleConns(maxIdle)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	// Create ACL (private) from stdlib db
	pgConn, err := NewPostgresqlClient(db, operationTimeout)
	if err != nil {
		db.Close()
		return nil, err
	}

	// Wrap ACL in repository (public port)
	return persistence.NewTimescaleRepository(pgConn)
}
