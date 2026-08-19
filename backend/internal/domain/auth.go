package domain

import (
	"context"
	"time"
)

type User struct {
	ID           string    `db:"id" json:"id"`
	Email        string    `db:"email" json:"email"`
	PasswordHash string    `db:"password_hash" json:"-"`
	Name         string    `db:"name" json:"name"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type Project struct {
	ID          string     `db:"id" json:"id"`
	Name        string     `db:"name" json:"name"`
	Description string     `db:"description" json:"description"`
	OwnerID     string     `db:"owner_id" json:"owner_id"`
	CreatedAt   time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

type ProjectMember struct {
	ID        string    `db:"id" json:"id"`
	ProjectID string    `db:"project_id" json:"project_id"`
	UserID    string    `db:"user_id" json:"user_id"`
	Role      string    `db:"role" json:"role"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type ProjectInvitation struct {
	ID        string    `db:"id" json:"id"`
	ProjectID string    `db:"project_id" json:"project_id"`
	Email     string    `db:"email" json:"email"`
	Role      string    `db:"role" json:"role"`
	InvitedBy string    `db:"invited_by" json:"invited_by"`
	Token     string    `db:"token" json:"token"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	ExpiresAt time.Time `db:"expires_at" json:"expires_at"`
}

type APIKey struct {
	ID         string     `db:"id" json:"id"`
	ProjectID  string     `db:"project_id" json:"project_id"`
	KeyHash    string     `db:"key_hash" json:"-"`
	KeyPrefix  string     `db:"key_prefix" json:"key_prefix"`
	Name       string     `db:"name" json:"name"`
	CreatedAt  time.Time  `db:"created_at" json:"created_at"`
	LastUsedAt *time.Time `db:"last_used_at" json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `db:"expires_at" json:"expires_at,omitempty"`
	RevokedAt  *time.Time `db:"revoked_at" json:"revoked_at,omitempty"`
}

type D1InfrastructureJob struct {
	ID             string    `db:"id" json:"id"`
	ProjectID      string    `db:"project_id" json:"project_id"`
	D1DatabaseUUID *string   `db:"d1_database_uuid" json:"d1_database_uuid,omitempty"`
	D1Name         string    `db:"d1_name" json:"d1_name"`
	Jurisdiction   string    `db:"jurisdiction" json:"jurisdiction"`
	Status         string    `db:"status" json:"status"`
	ErrorMessage   *string   `db:"error_message" json:"error_message,omitempty"`
	RetryCount     int       `db:"retry_count" json:"retry_count"`
	CreatedAt      time.Time `db:"created_at" json:"created_at"`
	UpdatedAt      time.Time `db:"updated_at" json:"updated_at"`
}

// AuthContext carries the authentication scope resolved from an API key.
// It is resolved at the MCP transport boundary and threaded through the
// service and repository layers so that every query is scoped to one project.
type AuthContext struct {
	ProjectID   string
	APIKeyID    string
	IsEncrypted bool
}

// Valid reports whether the auth context carries a usable project scope.
func (a AuthContext) Valid() bool {
	return a.ProjectID != ""
}

type AuthRepository interface {
	// User operations
	CreateUser(ctx context.Context, email, passwordHash, name string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id string) (*User, error)

	// Project operations
	CreateProject(ctx context.Context, name, description, ownerID string) (*Project, error)
	GetProjectByID(ctx context.Context, id string) (*Project, error)
	ListProjectsByOwnerID(ctx context.Context, ownerID string) ([]Project, error)
	SoftDeleteProject(ctx context.Context, id, ownerID string) error
	IsUserProjectMemberOrOwner(ctx context.Context, projectID, userID string) (bool, error)

	// D1 Provisioning Async Queue
	CreatePendingD1Infrastructure(ctx context.Context, projectID, dbName string) error
	GetPendingD1Jobs(ctx context.Context) ([]D1InfrastructureJob, error)
	UpdateD1InfrastructureStatus(ctx context.Context, projectID string, uuid *string, status string, errorMsg *string, retries int) error
	GetD1Status(ctx context.Context, projectID string) (*D1InfrastructureJob, error)

	// APIKey operations
	CreateAPIKey(ctx context.Context, projectID, name, keyHash, keyPrefix string, expiresAt *time.Time) (*APIKey, error)
	ListAPIKeysByProjectID(ctx context.Context, projectID string) ([]APIKey, error)
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*APIKey, error)
	UpdateAPIKeyLastUsed(ctx context.Context, id string) error
	RevokeAPIKey(ctx context.Context, id, projectID string) error
}
