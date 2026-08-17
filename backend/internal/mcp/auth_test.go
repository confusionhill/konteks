package mcp

import (
	"context"
	"errors"
	"testing"

	"github.com/dika/contextkit/backend/internal/domain"
)

type stubAuthRepo struct {
	keys map[string]*domain.APIKey
}

func (s *stubAuthRepo) GetAPIKeyByHash(ctx context.Context, keyHash string) (*domain.APIKey, error) {
	return s.keys[keyHash], nil
}

func (s *stubAuthRepo) UpdateAPIKeyLastUsed(ctx context.Context, id string) error {
	return nil
}

func TestAuthProvider_Authenticate_ValidKey(t *testing.T) {
	repo := &stubAuthRepo{keys: map[string]*domain.APIKey{
		HashAPIKey("ck_live_test"): {ID: "key-1", ProjectID: "proj-a"},
	}}
	p := NewAuthProvider(repo)
	auth, err := p.Authenticate(context.Background(), "ck_live_test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth.ProjectID != "proj-a" || auth.APIKeyID != "key-1" {
		t.Fatalf("unexpected auth: %+v", auth)
	}
}

func TestAuthProvider_Authenticate_InvalidKey(t *testing.T) {
	repo := &stubAuthRepo{keys: map[string]*domain.APIKey{}}
	p := NewAuthProvider(repo)
	_, err := p.Authenticate(context.Background(), "ck_live_nope")
	if !errors.Is(err, ErrInvalidAPIKey) {
		t.Fatalf("expected ErrInvalidAPIKey, got %v", err)
	}
}

func TestAuthProvider_Authenticate_MissingKey(t *testing.T) {
	repo := &stubAuthRepo{keys: map[string]*domain.APIKey{}}
	p := NewAuthProvider(repo)
	_, err := p.Authenticate(context.Background(), "")
	if !errors.Is(err, ErrMissingAPIKey) {
		t.Fatalf("expected ErrMissingAPIKey, got %v", err)
	}
}

func TestAuthProvider_CachesValidKey(t *testing.T) {
	p := NewAuthProvider(&stubAuthRepo{
		keys: map[string]*domain.APIKey{
			HashAPIKey("ck_live_test"): {ID: "key-1", ProjectID: "proj-a"},
		},
	})
	// Repeated authentication should keep yielding the same resolved context.
	if _, err := p.Authenticate(context.Background(), "ck_live_test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	auth2, err := p.Authenticate(context.Background(), "ck_live_test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth2.ProjectID != "proj-a" {
		t.Fatalf("unexpected auth: %+v", auth2)
	}
}

func TestAuthFromContext_RoundTrip(t *testing.T) {
	auth := domain.AuthContext{ProjectID: "proj-a", APIKeyID: "key-1"}
	ctx := WithAuth(context.Background(), auth)
	got, ok := AuthFromContext(ctx)
	if !ok || got.ProjectID != "proj-a" {
		t.Fatalf("auth did not round-trip: %+v ok=%v", got, ok)
	}
}

func TestAuthFromContext_EmptyNotValid(t *testing.T) {
	_, ok := AuthFromContext(context.Background())
	if ok {
		t.Fatal("empty context should not yield an auth")
	}
}