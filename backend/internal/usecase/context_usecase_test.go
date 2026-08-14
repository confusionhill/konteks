package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/dika/contextkit/backend/internal/config"
	"github.com/dika/contextkit/backend/internal/domain"
)

// Mock StorageProvider
type mockStorage struct{}

func (m *mockStorage) PutObject(ctx context.Context, key string, data []byte) (string, error) {
	return "s3://mock/" + key, nil
}
func (m *mockStorage) GetObject(ctx context.Context, key string) ([]byte, error) {
	return []byte("mocked"), nil
}
func (m *mockStorage) EnsureBucketExists(ctx context.Context) error {
	return nil
}
func (m *mockStorage) GeneratePresignedURL(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	return "url", nil
}

// Mock RoomRepository
type mockRoomRepo struct {
	rooms  map[string]*domain.Room
	states map[string]*domain.RoomState
}

func (m *mockRoomRepo) Create(ctx context.Context, projectID string, externalUserID string, description string, parentRoomID *string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy) (*domain.Room, error) {
	return nil, nil
}
func (m *mockRoomRepo) GetByID(ctx context.Context, projectID string, roomID string) (*domain.Room, error) {
	return m.rooms[roomID], nil
}
func (m *mockRoomRepo) Update(ctx context.Context, projectID string, roomID string, description *string, parentRoomID *string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy) (*domain.Room, error) {
	return nil, nil
}
func (m *mockRoomRepo) Delete(ctx context.Context, projectID string, roomID string) error {
	return nil
}
func (m *mockRoomRepo) GetState(ctx context.Context, roomID string) (*domain.RoomState, error) {
	return m.states[roomID], nil
}
func (m *mockRoomRepo) UpsertState(ctx context.Context, state *domain.RoomState) error {
	m.states[state.RoomID] = state
	return nil
}
func (m *mockRoomRepo) ListWithOverview(ctx context.Context, projectID string, search string, status string, sortBy string, sortOrder string, minMessages int, limit int, offset int) ([]domain.RoomOverview, int, error) {
	return nil, 0, nil
}
func (m *mockRoomRepo) ListChildRooms(ctx context.Context, projectID string, parentRoomID string) ([]domain.ChildAgentOverview, error) {
	return nil, nil
}
func (m *mockRoomRepo) ListRoomsForArchival(ctx context.Context, inactiveDuration time.Duration, roomID string) ([]domain.Room, error) {
	return nil, nil
}
func (m *mockRoomRepo) UpdateStatus(ctx context.Context, roomID string, status string) error {
	if r, ok := m.rooms[roomID]; ok {
		r.Status = status
	}
	return nil
}
func (m *mockRoomRepo) ArchiveRoomTx(ctx context.Context, roomID string, archiveS3Path string) error {
	return nil
}
func (m *mockRoomRepo) HydrateRoomTx(ctx context.Context, roomID string, state *domain.RoomState, messages []domain.Message, memories []domain.Memory) error {
	return nil
}

// Mock MessageRepository
type mockMessageRepo struct {
	messages map[string][]domain.Message
	maxSeq   int64
}

func (m *mockMessageRepo) Create(ctx context.Context, msg *domain.Message) error {
	return nil
}
func (m *mockMessageRepo) ListByRoom(ctx context.Context, roomID string, limit int, beforeSeq int64) ([]domain.Message, error) {
	return m.messages[roomID], nil
}
func (m *mockMessageRepo) GetMaxSequence(ctx context.Context, roomID string) (int64, error) {
	return m.maxSeq, nil
}
func (m *mockMessageRepo) ListProjectLogs(ctx context.Context, projectID string, roomID string, externalUserID string, search string, role string, sortBy string, sortOrder string, limit int, offset int) ([]domain.ProjectLogEntry, int, error) {
	return nil, 0, nil
}

// Mock MemoryRepository
type mockMemoryRepo struct{}

func (m *mockMemoryRepo) Create(ctx context.Context, memory *domain.Memory) error {
	return nil
}
func (m *mockMemoryRepo) SearchSemantic(ctx context.Context, roomID string, externalUserID *string, queryEmbedding []float32, freshnessWeight float64, limit int) ([]domain.Memory, error) {
	return nil, nil
}
func (m *mockMemoryRepo) ListByRoom(ctx context.Context, roomID string) ([]domain.Memory, error) {
	return nil, nil
}

// Mock EvaluationRepository
type mockEvalRepo struct{}

func (m *mockEvalRepo) Create(ctx context.Context, eval *domain.ContextEvaluation) error {
	return nil
}
func (m *mockEvalRepo) GetByID(ctx context.Context, evalID string) (*domain.ContextEvaluation, error) {
	return nil, nil
}
func (m *mockEvalRepo) GetByRoomID(ctx context.Context, roomID string) ([]domain.ContextEvaluation, error) {
	return nil, nil
}
func (m *mockEvalRepo) GetByProjectID(ctx context.Context, projectID string, limit int, offset int) ([]domain.ContextEvaluation, int, error) {
	return nil, 0, nil
}
func (m *mockEvalRepo) GetSummary(ctx context.Context, projectID string) (*domain.EvaluationSummary, error) {
	return &domain.EvaluationSummary{}, nil
}

// Mock OpenRouterServiceClient
type mockOpenRouter struct{}

func (m *mockOpenRouter) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}
func (m *mockOpenRouter) ExtractStateAndSummary(ctx context.Context, history []domain.Message, currentSummary string, schemaJSON string) (string, string, string, []string, error) {
	return "summary", "topic", "{}", []string{}, nil
}
func (m *mockOpenRouter) ResolveReferences(ctx context.Context, history []domain.Message, stateJSON string) (map[string]string, []domain.Entity, error) {
	return map[string]string{}, []domain.Entity{}, nil
}
func (m *mockOpenRouter) EvaluateContext(ctx context.Context, query string, contextPkg *domain.ContextPackage) (*domain.ContextEvaluation, error) {
	return &domain.ContextEvaluation{}, nil
}

func TestGetContextPackage_PendingDelta(t *testing.T) {
	roomID := "room-1"
	projectID := "project-1"

	room := &domain.Room{
		ID:        roomID,
		ProjectID: projectID,
		Status:    domain.RoomStatusActive,
	}

	state := &domain.RoomState{
		RoomID:            roomID,
		Summary:           "Testing Pending Delta",
		State:             json.RawMessage("{}"),
		LastSummarizedSeq: 5,
	}

	messages := []domain.Message{
		{RoomID: roomID, Content: "Msg 6", Sequence: 6},
		{RoomID: roomID, Content: "Msg 7", Sequence: 7},
		{RoomID: roomID, Content: "Msg 8", Sequence: 8},
	}

	roomRepo := &mockRoomRepo{
		rooms:  map[string]*domain.Room{roomID: room},
		states: map[string]*domain.RoomState{roomID: state},
	}

	messageRepo := &mockMessageRepo{
		messages: map[string][]domain.Message{roomID: messages},
		maxSeq:   8,
	}

	memoryRepo := &mockMemoryRepo{}
	openRouter := &mockOpenRouter{}
	storage := &mockStorage{}
	evalRepo := &mockEvalRepo{}
	cfg := &config.Config{EvaluationSampleRate: 0}

	u := NewContextUsecase(roomRepo, messageRepo, memoryRepo, openRouter, storage, evalRepo, nil, cfg)

	req := domain.GetContextRequest{
		Query: "hello",
	}

	pkg, err := u.GetContextPackage(context.Background(), projectID, roomID, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pkg == nil {
		t.Fatalf("expected context package to be non-nil")
	}

	if len(pkg.RelevantMessages) != 3 {
		t.Errorf("expected 3 relevant messages, got %d", len(pkg.RelevantMessages))
	}
}
