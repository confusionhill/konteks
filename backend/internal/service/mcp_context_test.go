package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dika/contextkit/backend/internal/domain"
)

type stubMessageStore struct {
	created *domain.Message
	list    []domain.Message
	total   int
	err     error
}

func (s *stubMessageStore) Create(ctx context.Context, msg *domain.Message) error {
	if s.err != nil {
		return s.err
	}
	s.created = msg
	return nil
}

func (s *stubMessageStore) ListByRoom(ctx context.Context, roomID string, limit int, beforeSeq int64) ([]domain.Message, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.list, nil
}

func (s *stubMessageStore) ListByRoomRange(ctx context.Context, roomID string, limit int, offset int) ([]domain.Message, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.list, nil
}

func (s *stubMessageStore) CountByRoom(ctx context.Context, roomID string) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.total, nil
}

type stubMemoryStore struct {
	searchResults []domain.Memory
	searchRoomID  string
	err           error
}

func (s *stubMemoryStore) Search(ctx context.Context, projectID string, roomID string, externalUserID *string, queryEmbedding []float32, freshnessWeight float64, limit int) ([]domain.Memory, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.searchRoomID = roomID
	return s.searchResults, nil
}

type stubEmbeddingClient struct {
	embedding []float32
	err       error
	texts     []string
}

func (c *stubEmbeddingClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	c.texts = append(c.texts, text)
	if c.err != nil {
		return nil, c.err
	}
	if c.embedding == nil {
		c.embedding = []float32{0.1, 0.2, 0.3}
	}
	return c.embedding, nil
}

func newContextTestService(room *domain.Room, state *domain.RoomState, msgs *stubMessageStore, mems *stubMemoryStore, emb *stubEmbeddingClient) *MCPContextService {
	rs := &stubRoomStore{room: room}
	ss := &stubStateStore{state: state}
	return NewMCPContextService(rs, ss, msgs, mems, emb)
}

func TestContextService_AddMessage_ValidatesContent(t *testing.T) {
	svc := newContextTestService(nil, nil, &stubMessageStore{}, &stubMemoryStore{}, &stubEmbeddingClient{})
	auth := domain.AuthContext{ProjectID: "proj-a"}
	_, err := svc.AddMessage(context.Background(), auth, "room-1", "user", "", false, nil)
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestContextService_AddMessage_ValidatesRole(t *testing.T) {
	svc := NewMCPContextService(
		&stubRoomStore{room: &domain.Room{ID: "room-1", ProjectID: "proj-a"}},
		&stubStateStore{},
		&stubMessageStore{},
		&stubMemoryStore{},
		&stubEmbeddingClient{},
	)
	auth := domain.AuthContext{ProjectID: "proj-a"}
	_, err := svc.AddMessage(context.Background(), auth, "room-1", "supervisor", "hi", false, nil)
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}

func TestContextService_AddMessage_Creates(t *testing.T) {
	msgs := &stubMessageStore{}
	svc := NewMCPContextService(
		&stubRoomStore{room: &domain.Room{ID: "room-1", ProjectID: "proj-a"}},
		&stubStateStore{},
		msgs,
		&stubMemoryStore{},
		&stubEmbeddingClient{},
	)
	auth := domain.AuthContext{ProjectID: "proj-a"}
	msg, err := svc.AddMessage(context.Background(), auth, "room-1", "User", "hello", false, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg.Role != "user" {
		t.Fatalf("role = %q, want normalized lowercase 'user'", msg.Role)
	}
	if msgs.created == nil || msgs.created.ID == "" {
		t.Fatal("message was not persisted")
	}
}

func TestContextService_AddMessage_RejectsMissingRoom(t *testing.T) {
	svc := newContextTestService(nil, nil, &stubMessageStore{}, &stubMemoryStore{}, &stubEmbeddingClient{})
	auth := domain.AuthContext{ProjectID: "proj-a"}
	_, err := svc.AddMessage(context.Background(), auth, "missing", "user", "hi", false, nil)
	if err == nil {
		t.Fatal("expected error for missing room")
	}
}

func TestContextService_GetContextPackage_AppliesDefaultsAndBudgets(t *testing.T) {
	room := &domain.Room{ID: "room-1", ProjectID: "proj-a", ExternalUserID: "user-42"}
	state := &domain.RoomState{
		RoomID:      "room-1",
		Summary:     "comparing hotels in bali",
		ActiveTopic: "hotel search",
		State:       json.RawMessage(`{"budget":1000}`),
	}
	var msgs []domain.Message
	for i := 0; i < 5; i++ {
		msgs = append(msgs, domain.Message{
			ID:       string(rune('a' + i)),
			RoomID:   "room-1",
			Role:     "user",
			Content:  strings.Repeat("x", 200),
			Sequence: int64(i + 1),
		})
	}
	embed := &stubEmbeddingClient{}
	svc := newContextTestService(room, state, &stubMessageStore{list: msgs}, &stubMemoryStore{}, embed)

	pkg, err := svc.GetContextPackage(context.Background(), domain.AuthContext{ProjectID: "proj-a"}, "room-1", domain.GetContextRequest{
		Query: "what did I say about budget",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pkg.RoomID != "room-1" {
		t.Fatalf("room_id = %q", pkg.RoomID)
	}
	if pkg.ActiveTopic != "hotel search" {
		t.Fatalf("active_topic = %q", pkg.ActiveTopic)
	}
	if len(embed.texts) != 1 {
		t.Fatalf("expected query embedding call, got %d", len(embed.texts))
	}
	if pkg.TokensEstimated <= 0 || pkg.TokensProcessed < pkg.TokensEstimated {
		t.Fatalf("token accounting off: estimated=%d processed=%d", pkg.TokensEstimated, pkg.TokensProcessed)
	}
}

func TestContextService_GetContextPackage_DoesNotEmbedInFastMode(t *testing.T) {
	room := &domain.Room{ID: "room-1", ProjectID: "proj-a"}
	state := &domain.RoomState{RoomID: "room-1", State: json.RawMessage(`{}`)}
	embed := &stubEmbeddingClient{}
	svc := newContextTestService(room, state, &stubMessageStore{}, &stubMemoryStore{}, embed)

	fm := true
	_, err := svc.GetContextPackage(context.Background(), domain.AuthContext{ProjectID: "proj-a"}, "room-1", domain.GetContextRequest{
		Query:  "anything",
		Policy: domain.ContextPolicy{FastMode: fm},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(embed.texts) != 0 {
		t.Fatalf("fast mode should skip embedding, called %d times", len(embed.texts))
	}
}

func TestContextService_GetContextPackage_RejectsMissingRoom(t *testing.T) {
	svc := newContextTestService(nil, nil, &stubMessageStore{}, &stubMemoryStore{}, &stubEmbeddingClient{})
	_, err := svc.GetContextPackage(context.Background(), domain.AuthContext{ProjectID: "proj-a"}, "missing", domain.GetContextRequest{})
	if err == nil {
		t.Fatal("expected error for missing room")
	}
}