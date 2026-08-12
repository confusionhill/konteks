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

func hasPronouns(query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	pronouns := []string{" it ", " that ", " which ", " this ", " them ", " option "}
	padded := " " + q + " "
	for _, p := range pronouns {
		if strings.Contains(padded, p) {
			return true
		}
	}
	// Short query under 3 words is highly likely to be a pronoun/follow-up reference
	if len(strings.Fields(q)) <= 3 && q != "" {
		return true
	}
	return false
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

	// 2. Merge Policy Options
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

	// Apply global defaults if still unset
	if policy.MaxTokens <= 0 {
		policy.MaxTokens = 4000
	}
	if policy.MaxHistoryDepth <= 0 {
		policy.MaxHistoryDepth = 50
	}
	if policy.FreshnessWeight <= 0 {
		policy.FreshnessWeight = 0.5
	}

	// 3. Determine Strategy
	strategy := policy.Strategy
	if strategy == "" || strategy == "auto" {
		strategy = classifyStrategy(req.Query)
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

	// Limit history by the MaxHistoryDepth policy constraint
	if msgLimit > policy.MaxHistoryDepth {
		msgLimit = policy.MaxHistoryDepth
	}

	// 4. Fetch RoomState
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

	// 5. Retrieve recent message history
	history, err := u.messageRepo.ListByRoom(ctx, roomID, msgLimit, 0)
	if err != nil {
		history = []domain.Message{}
	}

	// 6. Dynamic schema extraction override on-the-fly
	if len(req.DynamicSchema) > 0 && string(req.DynamicSchema) != "null" && string(req.DynamicSchema) != "{}" {
		log.Printf("[Context Engine]: dynamic schema provided. Performing on-the-fly state extraction...")
		newSummary, activeTopic, extractedState, errExtract := u.openRouterClient.ExtractStateAndSummary(ctx, history, state.Summary, string(req.DynamicSchema))
		if errExtract != nil {
			log.Printf("[Context Engine State Extraction Error]: %v", errExtract)
		} else {
			state.Summary = newSummary
			state.ActiveTopic = activeTopic
			state.State = json.RawMessage(extractedState)
		}
	}

	var parsedState any
	if err := json.Unmarshal(state.State, &parsedState); err != nil {
		parsedState = make(map[string]any)
	}

	// 7. Search semantic memories
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
			matchedMemories, errSearch = u.memoryRepo.SearchSemantic(ctx, roomID, extUserID, embedding, policy.FreshnessWeight, memLimit)
			if errSearch != nil {
				log.Printf("[Context Memory Search Error]: %v", errSearch)
			}
		}
	}

	// 8. Smart-Sync Reference Resolution (Pronoun check)
	references := make(map[string]string)
	entities := []domain.Entity{}
	if len(history) > 0 && hasPronouns(req.Query) {
		log.Printf("[Context Engine]: Pronoun detected in query. Resolving references...")
		var errRef error
		references, entities, errRef = u.openRouterClient.ResolveReferences(ctx, history, string(state.State))
		if errRef != nil {
			log.Printf("[Reference Resolution Warning]: %v", errRef)
			references = make(map[string]string)
			entities = []domain.Entity{}
		} else {
			// Persist resolved references & entities
			state.References = references
			state.Entities = entities
			_ = u.roomRepo.UpsertState(ctx, state)
		}
	} else if len(history) > 0 {
		// Use cached references if any
		references = state.References
		entities = state.Entities
		if references == nil {
			references = make(map[string]string)
		}
	}

	// 9. Greedy Token Packing
	budgetRemaining := policy.MaxTokens

	// Essential elements: Summary & State
	summaryTokens := len(state.Summary) / 4
	stateTokens := len(string(state.State)) / 4
	budgetRemaining -= (summaryTokens + stateTokens)
	if budgetRemaining < 0 {
		budgetRemaining = 0
	}

	// Pack messages from newest to oldest
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

	// Pack memories from highest ranking to lowest
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

	// Final estimated tokens sum
	estimatedTokens := summaryTokens + stateTokens
	for _, m := range packedMessages {
		estimatedTokens += len(m.Content) / 4
	}
	for _, mem := range packedMemories {
		estimatedTokens += len(mem.Content) / 4
	}

	// Tokens processed sum (everything scanned before truncation)
	processedTokens := summaryTokens + stateTokens
	for _, m := range history {
		processedTokens += len(m.Content) / 4
	}
	for _, mem := range matchedMemories {
		processedTokens += len(mem.Content) / 4
	}

	pkg := &domain.ContextPackage{
		RoomID:           roomID,
		Summary:          state.Summary,
		ActiveTopic:      state.ActiveTopic,
		State:            parsedState,
		RelevantMessages: packedMessages,
		Memories:         packedMemories,
		Entities:         entities,
		References:       references,
		StrategyUsed:     strategy,
		TokensEstimated:  estimatedTokens,
		TokensProcessed:  processedTokens,
	}

	return pkg, nil
}
