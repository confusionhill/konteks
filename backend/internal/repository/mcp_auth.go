package repository

import (
	"context"
	"database/sql"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

// MCPAuthRepository validates API keys for the MCP server. It is a thin,
// project-scoped lookup on the shared api_keys table.
type MCPAuthRepository struct {
	db *sqlx.DB
}

func NewMCPAuthRepository(db *sqlx.DB) *MCPAuthRepository {
	return &MCPAuthRepository{db: db}
}

// GetAPIKeyByHash returns the key if it exists and is active (not revoked, not
// expired). Returns nil if the key is unknown or inactive.
func (r *MCPAuthRepository) GetAPIKeyByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	query := `
		SELECT id, project_id, key_prefix, name, created_at, last_used_at, expires_at, revoked_at
		FROM api_keys
		WHERE key_hash = $1
		  AND revoked_at IS NULL
		  AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
	`
	var apiKey domain.APIKey
	if err := r.db.GetContext(ctx, &apiKey, query, keyHash); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &apiKey, nil
}

func (r *MCPAuthRepository) UpdateAPIKeyLastUsed(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE api_keys SET last_used_at = CURRENT_TIMESTAMP WHERE id = $1", id)
	return err
}