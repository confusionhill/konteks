package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

type authRepository struct {
	db *sqlx.DB
}

func NewAuthRepository(db *sqlx.DB) domain.AuthRepository {
	return &authRepository{db: db}
}

// CreateUser creates a new user record
func (r *authRepository) CreateUser(ctx context.Context, email, passwordHash, name string) (*domain.User, error) {
	query := `
		INSERT INTO users (email, password_hash, name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, name, created_at, updated_at
	`
	var user domain.User
	err := r.db.GetContext(ctx, &user, query, email, passwordHash, name)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail retrieves a user by email
func (r *authRepository) GetUserByEmail(ctx context.Context, email string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, name, created_at, updated_at
		FROM users
		WHERE email = $1
	`
	var user domain.User
	err := r.db.GetContext(ctx, &user, query, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetUserByID retrieves a user by ID
func (r *authRepository) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	query := `
		SELECT id, email, password_hash, name, created_at, updated_at
		FROM users
		WHERE id = $1
	`
	var user domain.User
	err := r.db.GetContext(ctx, &user, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// CreateProject creates a new project and inserts owner into project_members
func (r *authRepository) CreateProject(ctx context.Context, name, description, ownerID string) (*domain.Project, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	queryProj := `
		INSERT INTO projects (name, description, owner_id)
		VALUES ($1, $2, $3)
		RETURNING id, name, description, owner_id, created_at, updated_at, deleted_at
	`
	var proj domain.Project
	err = tx.GetContext(ctx, &proj, queryProj, name, description, ownerID)
	if err != nil {
		return nil, err
	}

	queryMember := `
		INSERT INTO project_members (project_id, user_id, role)
		VALUES ($1, $2, 'owner')
		ON CONFLICT (project_id, user_id) DO NOTHING
	`
	_, err = tx.ExecContext(ctx, queryMember, proj.ID, ownerID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &proj, nil
}

// GetProjectByID retrieves a non-deleted project by ID
func (r *authRepository) GetProjectByID(ctx context.Context, id string) (*domain.Project, error) {
	query := `
		SELECT id, name, description, owner_id, created_at, updated_at, deleted_at
		FROM projects
		WHERE id = $1 AND deleted_at IS NULL
	`
	var proj domain.Project
	err := r.db.GetContext(ctx, &proj, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &proj, nil
}

// ListProjectsByOwnerID lists all non-deleted projects where user is owner or member
func (r *authRepository) ListProjectsByOwnerID(ctx context.Context, ownerID string) ([]domain.Project, error) {
	query := `
		SELECT DISTINCT p.id, p.name, p.description, p.owner_id, p.created_at, p.updated_at, p.deleted_at
		FROM projects p
		LEFT JOIN project_members pm ON p.id = pm.project_id
		WHERE (p.owner_id = $1 OR pm.user_id = $1) AND p.deleted_at IS NULL
		ORDER BY p.created_at DESC
	`
	var projects []domain.Project
	err := r.db.SelectContext(ctx, &projects, query, ownerID)
	if err != nil {
		return nil, err
	}
	return projects, nil
}

// SoftDeleteProject soft-deletes a project by setting deleted_at
func (r *authRepository) SoftDeleteProject(ctx context.Context, id, ownerID string) error {
	query := `
		UPDATE projects
		SET deleted_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND owner_id = $2 AND deleted_at IS NULL
	`
	res, err := r.db.ExecContext(ctx, query, id, ownerID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("project not found or unauthorized to delete")
	}
	return nil
}

// IsUserProjectMemberOrOwner checks if a user owns or belongs to a non-deleted project
func (r *authRepository) IsUserProjectMemberOrOwner(ctx context.Context, projectID, userID string) (bool, error) {
	query := `
		SELECT 1
		FROM projects p
		LEFT JOIN project_members pm ON p.id = pm.project_id
		WHERE p.id = $1 AND (p.owner_id = $2 OR pm.user_id = $2) AND p.deleted_at IS NULL
		LIMIT 1
	`
	var exists int
	err := r.db.GetContext(ctx, &exists, query, projectID, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// CreateAPIKey generates an API key record for a project
func (r *authRepository) CreateAPIKey(ctx context.Context, projectID, name, keyHash, keyPrefix string, expiresAt *time.Time) (*domain.APIKey, error) {
	query := `
		INSERT INTO api_keys (project_id, name, key_hash, key_prefix, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, project_id, key_hash, key_prefix, name, created_at, last_used_at, expires_at, revoked_at
	`
	var apiKey domain.APIKey
	err := r.db.GetContext(ctx, &apiKey, query, projectID, name, keyHash, keyPrefix, expiresAt)
	if err != nil {
		return nil, err
	}
	return &apiKey, nil
}

// ListAPIKeysByProjectID lists non-revoked API keys for a project
func (r *authRepository) ListAPIKeysByProjectID(ctx context.Context, projectID string) ([]domain.APIKey, error) {
	query := `
		SELECT id, project_id, key_prefix, name, created_at, last_used_at, expires_at, revoked_at
		FROM api_keys
		WHERE project_id = $1
		ORDER BY created_at DESC
	`
	var keys []domain.APIKey
	err := r.db.SelectContext(ctx, &keys, query, projectID)
	if err != nil {
		return nil, err
	}
	return keys, nil
}

// GetAPIKeyByHash retrieves an API key by hash if active (not revoked, not expired)
func (r *authRepository) GetAPIKeyByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	query := `
		SELECT id, project_id, key_prefix, name, created_at, last_used_at, expires_at, revoked_at
		FROM api_keys
		WHERE key_hash = $1 
		  AND revoked_at IS NULL 
		  AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
	`
	var apiKey domain.APIKey
	err := r.db.GetContext(ctx, &apiKey, query, keyHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &apiKey, nil
}

// UpdateAPIKeyLastUsed updates last_used_at timestamp
func (r *authRepository) UpdateAPIKeyLastUsed(ctx context.Context, id string) error {
	query := `
		UPDATE api_keys
		SET last_used_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

// RevokeAPIKey marks an API key as revoked
func (r *authRepository) RevokeAPIKey(ctx context.Context, id, projectID string) error {
	query := `
		UPDATE api_keys
		SET revoked_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND project_id = $2 AND revoked_at IS NULL
	`
	res, err := r.db.ExecContext(ctx, query, id, projectID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("key not found or already revoked")
	}
	return nil
}
