package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"testing"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/dika/contextkit/backend/internal/worker"
)

type stubMemoryStoreAll struct {
	created     *domain.Memory
	searchRms   []domain.Memory
	listRms     []domain.Memory
	total       int
	deleteErr   error
	embeddingUp []string
}

func (s *stubMemoryStoreAll) Create(ctx context.Context, memory *domain.Memory) error {
	s.created = memory
	return nil
}

func (s *stubMemoryStoreAll) UpdateEmbedding(ctx context.Context, memoryID string, embedding []float32) error {
	s.embeddingUp = append(s.embeddingUp, memoryID)
	return nil
}

func (s *stubMemoryStoreAll) Search(ctx context.Context, projectID string, roomID string, externalUserID *string, queryEmbedding []float32, freshnessWeight float64, limit int) ([]domain.Memory, error) {
	return s.searchRms, nil
}

func (s *stubMemoryStoreAll) ListByRoom(ctx context.Context, projectID string, roomID string, limit int, offset int) ([]domain.Memory, int, error) {
	return s.listRms, s.total, nil
}

func (s *stubMemoryStoreAll) Delete(ctx context.Context, projectID string, memoryID string) error {
	return s.deleteErr
}

type stubEmbedder struct {
	tasks []worker.MCPMemoryEmbedTask
	err   error
}

func (e *stubEmbedder) Enqueue(task worker.MCPMemoryEmbedTask) error {
	if e.err != nil {
		return e.err
	}
	e.tasks = append(e.tasks, task)
	return nil
}

func newMemoryTestService(room *domain.Room, store *stubMemoryStoreAll, embed *stubEmbeddingClient, em *stubEmbedder) *MCPMemoryService {
	return NewMCPMemoryService(
		&stubRoomStore{room: room},
		store,
		embed,
		em,
	)
}

func TestMemoryService_AddMemory_RequiresContent(t *testing.T) {
	svc := newMemoryTestService(nil, &stubMemoryStoreAll{}, &stubEmbeddingClient{}, &stubEmbedder{})
	_, err := svc.AddMemory(context.Background(), domain.AuthContext{ProjectID: "proj-a"}, "room-1", "", nil, nil)
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

func TestMemoryService_AddMemory_PersistsAndQueues(t *testing.T) {
	room := &domain.Room{ID: "room-1", ProjectID: "proj-a", ExternalUserID: "user-42"}
	store := &stubMemoryStoreAll{}
	embed := &stubEmbeddingClient{}
	em := &stubEmbedder{}
	svc := newMemoryTestService(room, store, embed, em)

	mem, err := svc.AddMemory(context.Background(), domain.AuthContext{ProjectID: "proj-a"}, "room-1", "user prefers window seats", json.RawMessage(`{"type":"preference"}`), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.created == nil {
		t.Fatal("memory not persisted")
	}
	if store.created.ProjectID != "proj-a" {
		t.Fatalf("project = %q", store.created.ProjectID)
	}
	if store.created.ExternalUserID == nil || *store.created.ExternalUserID != "user-42" {
		t.Fatalf("external_user_id not inherited from room: %+v", store.created.ExternalUserID)
	}
	if mem.Metadata != `{"type":"preference"}` {
		t.Fatalf("metadata = %q", mem.Metadata)
	}
	if len(store.created.Embedding) != 0 {
		t.Fatal("memory should be persisted without embedding (async generation)")
	}
	if len(em.tasks) != 1 || em.tasks[0].MemoryID != mem.ID {
		t.Fatalf("expected one embedding task for %s, got %+v", mem.ID, em.tasks)
	}
}

func TestMemoryService_SearchMemories_EmbedsAndSearches(t *testing.T) {
	room := &domain.Room{ID: "room-1", ProjectID: "proj-a"}
	store := &stubMemoryStoreAll{}
	embed := &stubEmbeddingClient{}
	svc := newMemoryTestService(room, store, embed, &stubEmbedder{})

	mems, err := svc.SearchMemories(context.Background(), domain.AuthContext{ProjectID: "proj-a"}, "room-1", "budget hotels", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(embed.texts) != 1 || embed.texts[0] != "budget hotels" {
		t.Fatalf("query was not embedded, got %+v", embed.texts)
	}
	if mems == nil {
		t.Fatal("expected memory slice")
	}
}

func TestMemoryService_SearchMemories_RejectsEmptyQuery(t *testing.T) {
	svc := newMemoryTestService(&domain.Room{ID: "room-1", ProjectID: "proj-a"}, &stubMemoryStoreAll{}, &stubEmbeddingClient{}, &stubEmbedder{})
	_, err := svc.SearchMemories(context.Background(), domain.AuthContext{ProjectID: "proj-a"}, "room-1", "", 10)
	if err == nil {
		t.Fatal("expected error for empty query")
	}
}

func TestMemoryService_RemoveMemory_MissingRoom(t *testing.T) {
	store := &stubMemoryStoreAll{deleteErr: sql.ErrNoRows}
	svc := newMemoryTestService(&domain.Room{ID: "room-1", ProjectID: "proj-a"}, store, &stubEmbeddingClient{}, &stubEmbedder{})
	err := svc.RemoveMemory(context.Background(), domain.AuthContext{ProjectID: "proj-a"}, "mem-1")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("expected not-found-like error, got %v", err)
	}
}

func TestMemoryService_AddMemory_MissingRoom(t *testing.T) {
	svc := newMemoryTestService(nil, &stubMemoryStoreAll{}, &stubEmbeddingClient{}, &stubEmbedder{})
	_, err := svc.AddMemory(context.Background(), domain.AuthContext{ProjectID: "proj-a"}, "missing", "fact", nil, nil)
	if err == nil {
		t.Fatal("expected error for missing room")
	}
}