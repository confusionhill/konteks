package domain

import (
	"context"
)

// D1Store defines the interface for interacting with Cloudflare D1 via its HTTP API.
type D1Store interface {
	// Query executes a single SQL statement against the specified D1 database.
	Query(ctx context.Context, databaseUUID string, sql string, args ...any) ([]map[string]any, error)

	// Batch executes multiple SQL statements atomically in a single HTTP request.
	// Returns a slice of results, where each result is a slice of rows for that statement.
	Batch(ctx context.Context, databaseUUID string, queries []string, args [][]any) ([][]map[string]any, error)

	// CreateDatabase provisions a new Cloudflare D1 Database and returns its UUID.
	CreateDatabase(ctx context.Context, name string) (string, error)
}
