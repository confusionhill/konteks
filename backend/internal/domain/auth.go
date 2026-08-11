package domain

import "context"

type Tenant struct {
	ID        string `db:"id" json:"id"`
	Name      string `db:"name" json:"name"`
	CreatedAt string `db:"created_at" json:"created_at"`
}

type APIKey struct {
	ID        string `db:"id" json:"id"`
	TenantID  string `db:"tenant_id" json:"tenant_id"`
	KeyHash   string `db:"key_hash" json:"key_hash"`
	Name      string `db:"name" json:"name"`
	CreatedAt string `db:"created_at" json:"created_at"`
}

type AuthRepository interface {
	GetTenantByAPIKey(ctx context.Context, keyHash string) (*Tenant, error)
	CreateTenant(ctx context.Context, name string) (*Tenant, error)
	CreateAPIKey(ctx context.Context, tenantID string, name string, keyHash string) (*APIKey, error)
}
