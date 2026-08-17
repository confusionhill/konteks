package mcp

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
)

var (
	ErrMissingAPIKey = errors.New("missing API key")
	ErrInvalidAPIKey = errors.New("invalid, revoked, or expired API key")
)

// authCacheEntry memoizes API-key validation for a short window to avoid a
// database round-trip on every MCP request.
type authCacheEntry struct {
	auth      domain.AuthContext
	expiresAt time.Time
}

// apiKeyValidator is the API-key lookup contract used by AuthProvider. It is
// satisfied by repository.MCPAuthRepository.
type apiKeyValidator interface {
	GetAPIKeyByHash(ctx context.Context, keyHash string) (*domain.APIKey, error)
	UpdateAPIKeyLastUsed(ctx context.Context, id string) error
}

// AuthProvider validates the shared api_keys table and resolves an AuthContext
// ({project_id, api_key_id, is_encrypted}) for every tool invocation. API keys
// are identified by their SHA-256 hash, mirroring the main backend middleware.
type AuthProvider struct {
	authRepo apiKeyValidator
	mu       sync.Mutex
	cache    map[string]authCacheEntry
	ttl      time.Duration
}

func NewAuthProvider(authRepo apiKeyValidator) *AuthProvider {
	return &AuthProvider{
		authRepo: authRepo,
		cache:    make(map[string]authCacheEntry),
		ttl:      5 * time.Minute,
	}
}

// Authenticate hashes the raw key, looks it up, and builds a project-scoped
// AuthContext. It returns ErrInvalidAPIKey when the key is unknown, revoked, or
// expired. The last_used_at stamp is updated asynchronously.
func (p *AuthProvider) Authenticate(ctx context.Context, rawKey string) (domain.AuthContext, error) {
	if rawKey == "" {
		return domain.AuthContext{}, ErrMissingAPIKey
	}
	keyHash := HashAPIKey(rawKey)

	p.mu.Lock()
	if entry, ok := p.cache[keyHash]; ok && time.Now().Before(entry.expiresAt) {
		auth := entry.auth
		p.mu.Unlock()
		return auth, nil
	}
	p.mu.Unlock()

	apiKey, err := p.authRepo.GetAPIKeyByHash(ctx, keyHash)
	if err != nil {
		return domain.AuthContext{}, err
	}
	if apiKey == nil {
		return domain.AuthContext{}, ErrInvalidAPIKey
	}

	auth := domain.AuthContext{
		ProjectID: apiKey.ProjectID,
		APIKeyID:  apiKey.ID,
	}

	p.mu.Lock()
	p.cache[keyHash] = authCacheEntry{auth: auth, expiresAt: time.Now().Add(p.ttl)}
	p.mu.Unlock()

	go func(keyID string) {
		_ = p.authRepo.UpdateAPIKeyLastUsed(context.Background(), keyID)
	}(apiKey.ID)

	return auth, nil
}

// HashAPIKey returns the hex SHA-256 representation of a raw API key.
func HashAPIKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return fmt.Sprintf("%x", sum)
}