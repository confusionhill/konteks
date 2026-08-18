package usecase

import (
	"context"
	"testing"

	"github.com/dika/contextkit/backend/internal/domain"
)

type mockHealthRepo struct {
	policies map[string]*domain.ContextHealthPolicy
	memories map[string]*domain.Memory
	revisions []domain.MemoryRevision
}

func newMockHealthRepo() *mockHealthRepo {
	return &mockHealthRepo{
		policies: make(map[string]*domain.ContextHealthPolicy),
		memories: make(map[string]*domain.Memory),
	}
}

func (m *mockHealthRepo) GetProjectPolicy(ctx context.Context, projectID string) (*domain.ContextHealthPolicy, error) {
	if p, ok := m.policies[projectID]; ok {
		return p, nil
	}
	return &domain.ContextHealthPolicy{
		Enabled:             true,
		Mode:                domain.ContextHealthModeBalanced,
		MaxEntropyThreshold: 4.8,
		TurnEvalInterval:    50,
	}, nil
}

func (m *mockHealthRepo) UpdateProjectPolicy(ctx context.Context, projectID string, policy *domain.ContextHealthPolicy) error {
	m.policies[projectID] = policy
	return nil
}

func (m *mockHealthRepo) UpdateMemoryStatus(ctx context.Context, memoryID string, status string, supersededByID *string) error {
	if mem, ok := m.memories[memoryID]; ok {
		mem.Status = status
		mem.SupersededByID = supersededByID
	}
	return nil
}

func (m *mockHealthRepo) LogMemoryRevision(ctx context.Context, rev *domain.MemoryRevision) error {
	m.revisions = append(m.revisions, *rev)
	return nil
}

func (m *mockHealthRepo) ListMemoriesByRoom(ctx context.Context, roomID string, includeSuperseded bool) ([]domain.Memory, error) {
	var res []domain.Memory
	for _, mem := range m.memories {
		if mem.RoomID != nil && *mem.RoomID == roomID {
			if includeSuperseded || mem.Status == domain.MemoryStatusActive {
				res = append(res, *mem)
			}
		}
	}
	return res, nil
}

func (m *mockHealthRepo) RestoreMemory(ctx context.Context, memoryID string) error {
	if mem, ok := m.memories[memoryID]; ok {
		mem.Status = domain.MemoryStatusActive
		mem.SupersededByID = nil
	}
	return nil
}

func (m *mockHealthRepo) GetMemoryByID(ctx context.Context, memoryID string) (*domain.Memory, error) {
	return m.memories[memoryID], nil
}

func TestSanitizeIngressMemory(t *testing.T) {
	healthRepo := newMockHealthRepo()
	usecase := NewContextHealthUsecase(healthRepo, nil, nil, nil)

	// Whitelisted content (code block)
	codeContent := "```go\nfunc main() {}\n```"
	entropy, isWhitelisted, shouldFlag, err := usecase.SanitizeIngressMemory(context.Background(), "proj-1", codeContent)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isWhitelisted {
		t.Errorf("expected isWhitelisted = true for code block, got false")
	}
	if shouldFlag {
		t.Errorf("expected shouldFlag = false for whitelisted content, got true")
	}
	if entropy < 0 {
		t.Errorf("invalid entropy score: %f", entropy)
	}
}

func TestRestoreMemory(t *testing.T) {
	healthRepo := newMockHealthRepo()
	roomID := "room-1"
	projectID := "proj-1"

	memID := "mem-1"
	healthRepo.memories[memID] = &domain.Memory{
		ID:        memID,
		ProjectID: projectID,
		RoomID:    &roomID,
		Content:   "Old address: Tokyo",
		Status:    domain.MemoryStatusSuperseded,
	}

	usecase := NewContextHealthUsecase(healthRepo, nil, nil, nil)

	err := usecase.RestoreMemory(context.Background(), projectID, memID)
	if err != nil {
		t.Fatalf("unexpected error restoring memory: %v", err)
	}

	restoredMem := healthRepo.memories[memID]
	if restoredMem.Status != domain.MemoryStatusActive {
		t.Errorf("expected status 'active' after restore, got '%s'", restoredMem.Status)
	}
	if len(healthRepo.revisions) != 1 || healthRepo.revisions[0].Action != "restored" {
		t.Errorf("expected 1 'restored' revision log")
	}
}
