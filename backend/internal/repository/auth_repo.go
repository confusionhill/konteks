package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

type authRepository struct {
	db *sqlx.DB
}

func NewAuthRepository(db *sqlx.DB) domain.AuthRepository {
	return &authRepository{db: db}
}

func (r *authRepository) GetTenantByAPIKey(ctx context.Context, keyHash string) (*domain.Tenant, error) {
	query := `
		SELECT t.id, t.name, t.created_at
		FROM tenants t
		INNER JOIN api_keys k ON t.id = k.tenant_id
		WHERE k.key_hash = $1
	`
	var tenant domain.Tenant
	err := r.db.GetContext(ctx, &tenant, query, keyHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	return &tenant, nil
}

func (r *authRepository) CreateTenant(ctx context.Context, name string) (*domain.Tenant, error) {
	query := `
		INSERT INTO tenants (name)
		VALUES ($1)
		RETURNING id, name, created_at
	`
	var tenant domain.Tenant
	err := r.db.GetContext(ctx, &tenant, query, name)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

func (r *authRepository) CreateAPIKey(ctx context.Context, tenantID string, name string, keyHash string) (*domain.APIKey, error) {
	query := `
		INSERT INTO api_keys (tenant_id, name, key_hash)
		VALUES ($1, $2, $3)
		RETURNING id, tenant_id, key_hash, name, created_at
	`
	var apiKey domain.APIKey
	err := r.db.GetContext(ctx, &apiKey, query, tenantID, name, keyHash)
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}
