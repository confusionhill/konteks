package domain

import "context"

// D1Router defines the service responsible for mapping a project to its D1 database.
type D1Router interface {
	// Lookup returns the D1 database UUID for a given projectID.
	// It should use a fast cache (e.g., Redis) and fallback to the Postgres registry.
	Lookup(ctx context.Context, projectID string) (string, error)
}
