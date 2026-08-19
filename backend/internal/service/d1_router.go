package service

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

type d1Router struct {
	db    *sqlx.DB
	cache map[string]string // Simple in-memory cache for MVP (replace with Redis for production)
	mu    sync.RWMutex
}

// NewD1Router creates a new D1Router service.
func NewD1Router(db *sqlx.DB) domain.D1Router {
	return &d1Router{
		db:    db,
		cache: make(map[string]string),
	}
}

func (r *d1Router) Lookup(ctx context.Context, projectID string) (string, error) {
	// Check cache
	r.mu.RLock()
	if uuid, ok := r.cache[projectID]; ok {
		r.mu.RUnlock()
		return uuid, nil
	}
	r.mu.RUnlock()

	// Cache miss: query Postgres
	var uuid string
	query := "SELECT d1_database_uuid FROM project_d1_infrastructure WHERE project_id = $1"
	err := r.db.GetContext(ctx, &uuid, query, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("no D1 database provisioned for project %s", projectID)
		}
		return "", err
	}

	// Update cache
	r.mu.Lock()
	r.cache[projectID] = uuid
	r.mu.Unlock()

	return uuid, nil
}
