package persistence

import (
	"context"
)

// SQLConnection is a private anti-corruption layer interface.
// It abstracts the underlying SQL driver (stdlib sql.DB, pgx, etc.)
// and is only used internally by this adapter package.
//
// This is NOT a public port - it's an implementation detail.
// The public port is PositionRepository (in ports/).
type SQLConnection interface {
	// ExecInsert executes a parameterized INSERT statement.
	ExecInsert(ctx context.Context, query string, args ...interface{}) (rowsAffected int64, err error)

	// Close releases database resources.
	Close() error
}
