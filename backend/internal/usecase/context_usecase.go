package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/dika/contextkit/backend/internal/domain"
)

type contextUsecase struct {
	roomRepo         domain.RoomRepository
	messageRepo      domain.MessageRepository
	memoryRepo       domain.MemoryRepository
	openRouterClient domain.OpenRouterServiceClient
}

func NewContextUsecase(
	roomRepo domain.RoomRepository,
	messageRepo domain.MessageRepository,
	memoryRepo domain.MemoryRepository,
	openRouterClient domain.OpenRouterServiceClient,
) domain.ContextUsecase {
	return &contextUsecase{
		roomRepo:         roomRepo,
		messageRepo:      messageRepo,
		memoryRepo:       memoryRepo,
		openRouterClient: openRouterClient,
	}
}

// Zero-Latency Heuristic Intent Classifier (Sub-0.1ms Go string matching)
func classifyStrategy(query string) string {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return "hybrid"
	}

	// Rule 1: Recall intent -> memory_heavy
	recallKeywords := []string{"remember", "earlier", "previously", "last time", "past", "history", "preference", "my budget", "my favorite"}
	for _, kw := range recallKeywords {
		if strings.Contains(q, kw) {
			return "memory_heavy"
		}
	}

	// Rule 2: Follow-up / pronoun reference -> recent_heavy
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

	// Rule 3: Default -> hybrid
	return "hybrid"
}

func (u *contextUsecase) GetContextPackage(ctx context.Context, tenantID string, roomID string, req domain.GetContextRequest) (*domain.ContextPackage, error) {
	// 1. Verify room ownership
	room, err := u.roomRepo.GetByID(ctx, tenantID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	// 2. Determine retrieval strategy
	strategy := req.Policy.Strategy
	if strategy == "" || strategy == "auto" {
		strategy = classifyStrategy(req.Query)
	}

	// Set parameters based on strategy
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

	// 3. Fetch RoomState
	state, err := u.roomRepo.GetState(ctx, roomID)
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

	var parsedState any
	if err := json.Unmarshal(state.State, &parsedState); err != nil {
		parsedState = make(map[string]any)
	}

	// 4. Retrieve recent message history
	history, err := u.messageRepo.ListByRoom(ctx, roomID, msgLimit, 0)
	if err != nil {
		history = []domain.Message{}
	}

	// 5. Search semantic memories if query is provided
	var matchedMemories []domain.Memory
	if req.Query != "" {
		embedding, errEmbedding := u.openRouterClient.GenerateEmbedding(ctx, req.Query)
		if errEmbedding != nil {
			log.Printf("[Context Query Embedding Error]: %v", errEmbedding)
		} else {
			var extUserID *string
			if room.ExternalUserID != "" {
				extUserID = &room.ExternalUserID
			}
			var errSearch error
			matchedMemories, errSearch = u.memoryRepo.SearchSemantic(ctx, roomID, extUserID, embedding, memLimit)
			if errSearch != nil {
				log.Printf("[Context Memory Search Error]: %v", errSearch)
			}
		}
	}

	// 6. Perform Reference Resolution via OpenRouter (DeepSeek) if history exists
	references := make(map[string]string)
	entities := []domain.Entity{}
	if len(history) > 0 {
		var errRef error
		references, entities, errRef = u.openRouterClient.ResolveReferences(ctx, history, string(state.State))
		if errRef != nil {
			log.Printf("[Reference Resolution Warning]: %v", errRef)
			references = make(map[string]string)
			entities = []domain.Entity{}
		} else {
			// Persist resolved references & entities to DB so Dashboard can read them with 0 LLM cost
			state.References = references
			state.Entities = entities
			_ = u.roomRepo.UpsertState(ctx, state)
		}
	}

	// 7. Token Budget Allocator & Policy Enforcer
	maxTokens := req.Policy.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 4000 // default budget cap
	}

	// Rough token estimation: char_length / 4
	estimatedTokens := (len(state.Summary) + len(string(state.State))) / 4
	for _, m := range history {
		estimatedTokens += len(m.Content) / 4
	}
	for _, mem := range matchedMemories {
		estimatedTokens += len(mem.Content) / 4
	}

	// Truncate lowest-scoring memories if budget exceeded
	for estimatedTokens > maxTokens && len(matchedMemories) > 0 {
		removed := matchedMemories[len(matchedMemories)-1]
		matchedMemories = matchedMemories[:len(matchedMemories)-1]
		estimatedTokens -= len(removed.Content) / 4
	}

	pkg := &domain.ContextPackage{
		RoomID:           roomID,
		Summary:          state.Summary,
		ActiveTopic:      state.ActiveTopic,
		State:            parsedState,
		RelevantMessages: history,
		Memories:         matchedMemories,
		Entities:         entities,
		References:       references,
		StrategyUsed:     strategy,
		TokensEstimated:  estimatedTokens,
	}

	return pkg, nil
}
