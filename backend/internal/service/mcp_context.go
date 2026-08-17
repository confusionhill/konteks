package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/google/uuid"
)

// messageStore is the message persistence contract for the MCP context service.
type messageStore interface {
	Create(ctx context.Context, msg *domain.Message) error
	ListByRoom(ctx context.Context, roomID string, limit int, beforeSeq int64) ([]domain.Message, error)
	ListByRoomRange(ctx context.Context, roomID string, limit int, offset int) ([]domain.Message, error)
	CountByRoom(ctx context.Context, roomID string) (int, error)
}

// memorySearchStore is the memory search contract for context assembly.
type memorySearchStore interface {
	Search(ctx context.Context, projectID string, roomID string, externalUserID *string, queryEmbedding []float32, freshnessWeight float64, limit int) ([]domain.Memory, error)
}

// embeddingClient generates embeddings for context queries.
type embeddingClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
}

// MCPContextService assembles context packages and manages messages for the
// MCP server. It mirrors the main backend's context engine without the
// evaluation worker, artifact retrieval, or archive hydration dependencies.
type MCPContextService struct {
	roomRepo    roomStore
	stateRepo   stateStore
	messageRepo messageStore
	memoryRepo  memorySearchStore
	openRouter  embeddingClient
}

func NewMCPContextService(
	roomRepo roomStore,
	stateRepo stateStore,
	messageRepo messageStore,
	memoryRepo memorySearchStore,
	openRouter embeddingClient,
) *MCPContextService {
	return &MCPContextService{
		roomRepo:    roomRepo,
		stateRepo:   stateRepo,
		messageRepo: messageRepo,
		memoryRepo:  memoryRepo,
		openRouter:  openRouter,
	}
}

func (s *MCPContextService) AddMessage(ctx context.Context, auth domain.AuthContext, roomID string, role string, content string, isEncrypted bool, encryptionMetadata *string) (*domain.Message, error) {
	if content == "" {
		return nil, fmt.Errorf("content is required")
	}
	allowedRoles := map[string]bool{"user": true, "assistant": true, "system": true, "tool": true}
	if !allowedRoles[strings.ToLower(role)] {
		return nil, fmt.Errorf("invalid role %q: must be one of user, assistant, system, tool", role)
	}

	room, err := s.roomRepo.GetByID(ctx, auth.ProjectID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	msg := &domain.Message{
		ID:                 uuid.New().String(),
		RoomID:             roomID,
		Role:               strings.ToLower(role),
		Content:            content,
		IsEncrypted:        isEncrypted,
		EncryptionMetadata: encryptionMetadata,
	}
	if err := s.messageRepo.Create(ctx, msg); err != nil {
		return nil, err
	}
	return msg, nil
}

func (s *MCPContextService) ListMessages(ctx context.Context, auth domain.AuthContext, roomID string, limit int, offset int) ([]domain.Message, int, error) {
	room, err := s.roomRepo.GetByID(ctx, auth.ProjectID, roomID)
	if err != nil {
		return nil, 0, err
	}
	if room == nil {
		return nil, 0, fmt.Errorf("room not found or unauthorized")
	}
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	total, err := s.messageRepo.CountByRoom(ctx, roomID)
	if err != nil {
		return nil, 0, err
	}
	messages, err := s.messageRepo.ListByRoomRange(ctx, roomID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	return messages, total, nil
}

// GetContextPackage assembles a token-budgeted context package for a room.
// The optional query drives semantic memory retrieval; strategy and policy
// defaults mirror the main backend engine.
func (s *MCPContextService) GetContextPackage(ctx context.Context, auth domain.AuthContext, roomID string, req domain.GetContextRequest) (*domain.ContextPackage, error) {
	room, err := s.roomRepo.GetByID(ctx, auth.ProjectID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	policy := req.Policy
	if room.DefaultPolicy != nil {
		if policy.MaxTokens <= 0 {
			policy.MaxTokens = room.DefaultPolicy.MaxTokens
		}
		if policy.MaxHistoryDepth <= 0 {
			policy.MaxHistoryDepth = room.DefaultPolicy.MaxHistoryDepth
		}
		if policy.FreshnessWeight <= 0 {
			policy.FreshnessWeight = room.DefaultPolicy.FreshnessWeight
		}
		if policy.Strategy == "" {
			policy.Strategy = room.DefaultPolicy.Strategy
		}
		if policy.MinRelevanceScore <= 0 {
			policy.MinRelevanceScore = room.DefaultPolicy.MinRelevanceScore
		}
	}

	if policy.MaxTokens <= 0 {
		policy.MaxTokens = 4000
	}
	if policy.MaxHistoryDepth <= 0 {
		policy.MaxHistoryDepth = 50
	}
	if policy.FreshnessWeight <= 0 {
		policy.FreshnessWeight = 0.5
	}

	strategy := policy.Strategy
	if strategy == "" || strategy == "auto" {
		strategy = classifyMemoryStrategy(req.Query)
	}

	msgLimit := 10
	memLimit := 5
	switch strategy {
	case "recent_heavy":
		msgLimit = 25
		memLimit = 2
	case "memory_heavy":
		msgLimit = 5
		memLimit = 15
	case "hybrid":
		msgLimit = 10
		memLimit = 5
	}
	if msgLimit > policy.MaxHistoryDepth {
		msgLimit = policy.MaxHistoryDepth
	}

	state, err := s.stateRepo.Get(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = &domain.RoomState{
			RoomID:      roomID,
			Summary:     "",
			ActiveTopic: "",
			State:       json.RawMessage("{}"),
		}
	}

	history, err := s.messageRepo.ListByRoom(ctx, roomID, msgLimit, 0)
	if err != nil {
		return nil, err
	}

	references := make(map[string]string)
	entities := []domain.Entity{}

	var matchedMemories []domain.Memory

	if !policy.FastMode {
		if req.Query != "" {
			queryToEmbed := req.Query
			if len(queryToEmbed) > 1000 {
				queryToEmbed = queryToEmbed[:1000]
			}
			embedCtx, cancel := context.WithTimeout(ctx, 2500*time.Millisecond)
			defer cancel()
			embedding, errEmb := s.openRouter.GenerateEmbedding(embedCtx, queryToEmbed)
			if errEmb != nil {
				log.Printf("[MCP Context Embedding Error]: %v", errEmb)
			} else {
				var extUserID *string
				if room.ExternalUserID != "" {
					extUserID = &room.ExternalUserID
				}
				memSearch, errSearch := s.memoryRepo.Search(ctx, auth.ProjectID, roomID, extUserID, embedding, policy.FreshnessWeight, memLimit)
				if errSearch != nil {
					log.Printf("[MCP Context Memory Search Error]: %v", errSearch)
				} else {
					matchedMemories = memSearch
				}
			}
		}
	} else if len(history) > 0 {
		references = state.References
		entities = state.Entities
		if references == nil {
			references = make(map[string]string)
		}
		if entities == nil {
			entities = []domain.Entity{}
		}
	}

	var parsedState any
	if err := json.Unmarshal(state.State, &parsedState); err != nil {
		parsedState = make(map[string]any)
	}

	budgetRemaining := policy.MaxTokens
	summaryTokens := len(state.Summary) / 4
	stateTokens := len(string(state.State)) / 4
	budgetRemaining -= (summaryTokens + stateTokens)
	if budgetRemaining < 0 {
		budgetRemaining = 0
	}

	var packedMessages []domain.Message
	for i := len(history) - 1; i >= 0; i-- {
		msgTokens := len(history[i].Content) / 4
		if budgetRemaining >= msgTokens {
			packedMessages = append([]domain.Message{history[i]}, packedMessages...)
			budgetRemaining -= msgTokens
		} else {
			break
		}
	}

	var packedMemories []domain.Memory
	for _, mem := range matchedMemories {
		memTokens := len(mem.Content) / 4
		if budgetRemaining >= memTokens {
			packedMemories = append(packedMemories, mem)
			budgetRemaining -= memTokens
		} else {
			break
		}
	}

	estimatedTokens := summaryTokens + stateTokens
	for _, m := range packedMessages {
		estimatedTokens += len(m.Content) / 4
	}
	for _, mem := range packedMemories {
		estimatedTokens += len(mem.Content) / 4
	}

	processedTokens := summaryTokens + stateTokens
	for _, m := range history {
		processedTokens += len(m.Content) / 4
	}
	for _, mem := range matchedMemories {
		processedTokens += len(mem.Content) / 4
	}

	var childAgents []domain.ChildAgentOverview
	if req.IncludeChildren {
		children, errChild := s.roomRepo.ListChildRooms(ctx, auth.ProjectID, roomID)
		if errChild == nil {
			childAgents = children
		} else {
			childAgents = []domain.ChildAgentOverview{}
		}
	}

	return &domain.ContextPackage{
		RoomID:           roomID,
		Summary:          state.Summary,
		ActiveTopic:      state.ActiveTopic,
		State:            parsedState,
		RelevantMessages: packedMessages,
		Memories:         packedMemories,
		Entities:         entities,
		References:       references,
		ChildAgents:      childAgents,
		StrategyUsed:     strategy,
		TokensEstimated:  estimatedTokens,
		TokensProcessed:  processedTokens,
	}, nil
}

func classifyMemoryStrategy(query string) string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return "hybrid"
	}
	recallKeywords := []string{"remember", "earlier", "previously", "last time", "past", "history", "preference", "my budget", "my favorite"}
	for _, kw := range recallKeywords {
		if strings.Contains(q, kw) {
			return "memory_heavy"
		}
	}
	words := strings.Fields(q)
	if len(words) < 5 {
		return "recent_heavy"
	}
	pronouns := []string{" it ", " that ", " which ", " this ", " them ", " the hotel ", " the flight ", " option "}
	padded := " " + q + " "
	for _, p := range pronouns {
		if strings.Contains(padded, p) {
			return "recent_heavy"
		}
	}
	return "hybrid"
}