package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/dika/contextkit/backend/internal/worker"
	"github.com/google/uuid"
)

// memoryStore is the memory persistence contract for the MCP memory service.
type memoryStore interface {
	Create(ctx context.Context, memory *domain.Memory) error
	UpdateEmbedding(ctx context.Context, memoryID string, embedding []float32) error
	Search(ctx context.Context, projectID string, roomID string, externalUserID *string, queryEmbedding []float32, freshnessWeight float64, limit int) ([]domain.Memory, error)
	ListByRoom(ctx context.Context, projectID string, roomID string, limit int, offset int) ([]domain.Memory, int, error)
	Delete(ctx context.Context, projectID string, memoryID string) error
}

// memoryEmbedder asynchronously generates embeddings for stored memories.
type memoryEmbedder interface {
	Enqueue(worker.MCPMemoryEmbedTask) error
}

// MCPMemoryService manages memories and semantic search for the MCP server.
// add_memory persists immediately with a NULL embedding and queues the embedding
// generation to the async worker pool, so the tool never blocks on the LLM.
type MCPMemoryService struct {
	roomRepo   roomStore
	memoryRepo memoryStore
	openRouter embeddingClient
	embedder   memoryEmbedder
}

func NewMCPMemoryService(
	roomRepo roomStore,
	memoryRepo memoryStore,
	openRouter embeddingClient,
	embedder memoryEmbedder,
) *MCPMemoryService {
	return &MCPMemoryService{
		roomRepo:   roomRepo,
		memoryRepo: memoryRepo,
		openRouter: openRouter,
		embedder:   embedder,
	}
}

func (s *MCPMemoryService) AddMemory(ctx context.Context, auth domain.AuthContext, roomID string, content string, metadata json.RawMessage, externalUserID *string) (*domain.Memory, error) {
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	room, err := s.roomRepo.GetByID(ctx, auth.ProjectID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	metadataStr := "{}"
	if len(metadata) > 0 && string(metadata) != "null" && string(metadata) != "{}" {
		metadataStr = string(metadata)
	}

	var userID *string
	if externalUserID != nil && *externalUserID != "" {
		userID = externalUserID
	} else if room.ExternalUserID != "" {
		userID = &room.ExternalUserID
	}

	mem := &domain.Memory{
		ID:             uuid.New().String(),
		ProjectID:      auth.ProjectID,
		RoomID:         &roomID,
		ExternalUserID: userID,
		Content:        content,
		Metadata:       metadataStr,
		Status:         "active",
	}
	if err := s.memoryRepo.Create(ctx, mem); err != nil {
		return nil, err
	}

	err = s.embedder.Enqueue(worker.MCPMemoryEmbedTask{
		ProjectID: auth.ProjectID,
		MemoryID:  mem.ID,
		Content:   content,
	})
	if err != nil {
		log.Printf("[MCP Memory Service Warning]: failed to queue embedding for memory %s: %v", mem.ID, err)
	}
	return mem, nil
}

func (s *MCPMemoryService) SearchMemories(ctx context.Context, auth domain.AuthContext, roomID string, query string, limit int) ([]domain.Memory, error) {
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	room, err := s.roomRepo.GetByID(ctx, auth.ProjectID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}
	if limit <= 0 {
		limit = 10
	}

	if len(query) > 1000 {
		query = query[:1000]
	}
	embedCtx, cancel := context.WithTimeout(ctx, 2500*time.Millisecond)
	defer cancel()
	embedding, errEmbed := s.openRouter.GenerateEmbedding(embedCtx, query)
	if errEmbed != nil {
		if strings.Contains(errEmbed.Error(), "context deadline exceeded") || strings.Contains(errEmbed.Error(), "deadline exceeded") {
			return nil, fmt.Errorf("semantic search timed out generating query embedding")
		}
		return nil, fmt.Errorf("failed to generate query embedding: %w", errEmbed)
	}

	var extUserID *string
	if room.ExternalUserID != "" {
		extUserID = &room.ExternalUserID
	}
	memories, err := s.memoryRepo.Search(ctx, auth.ProjectID, roomID, extUserID, embedding, 0.5, limit)
	if err != nil {
		return nil, err
	}
	if memories == nil {
		memories = []domain.Memory{}
	}
	return memories, nil
}

func (s *MCPMemoryService) ListMemories(ctx context.Context, auth domain.AuthContext, roomID string, limit int, offset int) ([]domain.Memory, int, error) {
	room, err := s.roomRepo.GetByID(ctx, auth.ProjectID, roomID)
	if err != nil {
		return nil, 0, err
	}
	if room == nil {
		return nil, 0, fmt.Errorf("room not found or unauthorized")
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.memoryRepo.ListByRoom(ctx, auth.ProjectID, roomID, limit, offset)
}

func (s *MCPMemoryService) RemoveMemory(ctx context.Context, auth domain.AuthContext, memoryID string) error {
	if memoryID == "" {
		return fmt.Errorf("memory_id is required")
	}
	err := s.memoryRepo.Delete(ctx, auth.ProjectID, memoryID)
	if err == sql.ErrNoRows {
		return fmt.Errorf("memory not found or unauthorized")
	}
	return err
}