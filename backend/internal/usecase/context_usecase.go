package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"time"
	"math/rand"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/dika/contextkit/backend/internal/config"

	"github.com/dika/contextkit/backend/internal/domain"
)

type contextUsecase struct {
	roomRepo         domain.RoomRepository
	messageRepo      domain.MessageRepository
	memoryRepo       domain.MemoryRepository
	openRouterClient domain.OpenRouterServiceClient
	storageProvider  domain.StorageProvider
	evalRepo         domain.EvaluationRepository
	cfg              *config.Config
	evalQueue        chan evalTask
}

type evalTask struct {
	Query string
	Pkg   *domain.ContextPackage
}

func NewContextUsecase(
	roomRepo domain.RoomRepository,
	messageRepo domain.MessageRepository,
	memoryRepo domain.MemoryRepository,
	openRouterClient domain.OpenRouterServiceClient,
	storageProvider domain.StorageProvider,
	evalRepo domain.EvaluationRepository,
	cfg *config.Config,
) domain.ContextUsecase {
	u := &contextUsecase{
		roomRepo:         roomRepo,
		messageRepo:      messageRepo,
		memoryRepo:       memoryRepo,
		openRouterClient: openRouterClient,
		storageProvider:  storageProvider,
		evalRepo:         evalRepo,
		cfg:              cfg,
		evalQueue:        make(chan evalTask, 1000),
	}
	
	// Spawn background evaluation workers
	for i := 0; i < 2; i++ {
		go u.evaluationWorker()
	}

	return u
}

func (u *contextUsecase) evaluationWorker() {
	for task := range u.evalQueue {
		ctx := context.Background()
		eval, err := u.openRouterClient.EvaluateContext(ctx, task.Query, task.Pkg)
		if err != nil {
			log.Printf("[Evaluation Worker Error]: failed to evaluate context: %v", err)
			continue
		}
		
		eval.ID = uuid.NewString()
		eval.RoomID = task.Pkg.RoomID
		if err := u.evalRepo.Create(ctx, eval); err != nil {
			log.Printf("[Evaluation Worker Error]: failed to save evaluation: %v", err)
		} else {
			log.Printf("[Evaluation Worker]: successfully evaluated context for room %s (Score: %.2f/%.2f)", eval.RoomID, eval.RelevanceScore, eval.CompletenessScore)
		}
	}
}

type ArchivedRoomPayload struct {
	RoomState domain.RoomState `json:"state"`
	Messages  []domain.Message `json:"messages"`
	Memories  []domain.Memory  `json:"memories"`
}

func decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
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

	// Trigger Stage 1 Hydration if Room is Archived
	if room.Status == domain.RoomStatusArchived {
		log.Printf("[Context Engine]: Room %s is archived. Triggering Stage 1 Hydration...", roomID)
		err = u.roomRepo.UpdateStatus(ctx, roomID, domain.RoomStatusHydrating)
		if err != nil {
			return nil, fmt.Errorf("failed to lock room status to hydrating: %w", err)
		}

		if room.ArchiveS3Path == nil || *room.ArchiveS3Path == "" {
			_ = u.roomRepo.UpdateStatus(ctx, roomID, domain.RoomStatusArchived)
			return nil, fmt.Errorf("room is archived but archive S3 path is empty")
		}

		archiveBytes, err := u.storageProvider.GetObject(ctx, *room.ArchiveS3Path)
		if err != nil {
			_ = u.roomRepo.UpdateStatus(ctx, roomID, domain.RoomStatusArchived)
			return nil, fmt.Errorf("failed to retrieve room archive: %w", err)
		}

		decompressed, err := decompressGzip(archiveBytes)
		if err != nil {
			_ = u.roomRepo.UpdateStatus(ctx, roomID, domain.RoomStatusArchived)
			return nil, fmt.Errorf("failed to decompress archive data: %w", err)
		}

		var payload ArchivedRoomPayload
		if err := json.Unmarshal(decompressed, &payload); err != nil {
			_ = u.roomRepo.UpdateStatus(ctx, roomID, domain.RoomStatusArchived)
			return nil, fmt.Errorf("failed to parse archive JSON: %w", err)
		}

		// Stage 1: Synchronously restore RoomState and last 3 messages
		var stage1Msgs []domain.Message
		var stage2Msgs []domain.Message
		if len(payload.Messages) > 3 {
			stage1Msgs = payload.Messages[len(payload.Messages)-3:]
			stage2Msgs = payload.Messages[:len(payload.Messages)-3]
		} else {
			stage1Msgs = payload.Messages
		}

		err = u.roomRepo.HydrateRoomTx(ctx, roomID, &payload.RoomState, stage1Msgs, nil)
		if err != nil {
			_ = u.roomRepo.UpdateStatus(ctx, roomID, domain.RoomStatusArchived)
			return nil, fmt.Errorf("failed to execute Stage 1 hydration: %w", err)
		}

		// Stage 2: Asynchronously backfill the rest in a background goroutine
		go func() {
			bgCtx := context.Background()
			log.Printf("[Context Engine]: Spawning Stage 2 Backfill for room %s...", roomID)
			errHydrate := u.roomRepo.HydrateRoomTx(bgCtx, roomID, nil, stage2Msgs, payload.Memories)
			if errHydrate != nil {
				log.Printf("[Context Engine Background Hydration Error]: %v", errHydrate)
			} else {
				log.Printf("[Context Engine]: Stage 2 Backfill completed successfully for room %s.", roomID)
			}
		}()

		room.Status = domain.RoomStatusActive
	}

	// If room is hydrating, wait/block until active
	if room.Status == domain.RoomStatusHydrating {
		log.Printf("[Context Engine]: Room %s is currently hydrating. Blocking request...", roomID)
		start := time.Now()
		for {
			time.Sleep(50 * time.Millisecond)
			r, err := u.roomRepo.GetByID(ctx, tenantID, roomID)
			if err == nil && r != nil && r.Status == domain.RoomStatusActive {
				room = r
				break
			}
			if time.Since(start) > 5*time.Second {
				return nil, fmt.Errorf("timeout waiting for room hydration")
			}
		}
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

	// 5. Retrieve recent message history (Pending Delta Dynamic Fetching)
	maxSeq, errMaxSeq := u.messageRepo.GetMaxSequence(ctx, roomID)
	if errMaxSeq != nil {
		maxSeq = 0
	}
	pendingCount := maxSeq - state.LastSummarizedSeq
	finalMsgLimit := msgLimit
	if pendingCount > int64(finalMsgLimit) {
		finalMsgLimit = int(pendingCount)
	}

	history, err := u.messageRepo.ListByRoom(ctx, roomID, finalMsgLimit, 0)
	if err != nil {
		history = []domain.Message{}
	}

	// 6-8. Execute LLM dependencies (Extraction, Embeddings, Pronouns)
	var matchedMemories []domain.Memory
	references := make(map[string]string)
	entities := []domain.Entity{}

	if !policy.FastMode {
		var g errgroup.Group

		var extractSuccess bool
		var newSummary, activeTopic string
		var extractedState []byte

		// 6. Dynamic schema extraction override on-the-fly
		if len(req.DynamicSchema) > 0 && string(req.DynamicSchema) != "null" && string(req.DynamicSchema) != "{}" {
			g.Go(func() error {
				log.Printf("[Context Engine]: dynamic schema provided. Performing on-the-fly state extraction...")
				sum, topic, extState, errExtract := u.openRouterClient.ExtractStateAndSummary(ctx, history, state.Summary, string(req.DynamicSchema))
				if errExtract != nil {
					log.Printf("[Context Engine State Extraction Error]: %v", errExtract)
				} else {
					newSummary = sum
					activeTopic = topic
					extractedState = []byte(extState)
					extractSuccess = true
				}
				return nil
			})
		}

		// 7. Search semantic memories
		if req.Query != "" {
			g.Go(func() error {
				embedding, errEmbedding := u.openRouterClient.GenerateEmbedding(ctx, req.Query)
				if errEmbedding != nil {
					log.Printf("[Context Query Embedding Error]: %v", errEmbedding)
				} else {
					var extUserID *string
					if room.ExternalUserID != "" {
						extUserID = &room.ExternalUserID
					}
					var errSearch error
					memSearch, errSearch := u.memoryRepo.SearchSemantic(ctx, roomID, extUserID, embedding, policy.FreshnessWeight, memLimit)
					if errSearch != nil {
						log.Printf("[Context Memory Search Error]: %v", errSearch)
					} else {
						matchedMemories = memSearch
					}
				}
				return nil
			})
		}

		// 8. Smart-Sync Reference Resolution (Pronoun check)
		var refSuccess bool
		var resolvedRefs map[string]string
		var resolvedEnts []domain.Entity

		if len(history) > 0 && hasPronouns(req.Query) {
			currentStateStr := string(state.State)
			g.Go(func() error {
				log.Printf("[Context Engine]: Pronoun detected in query. Resolving references...")
				refs, ents, errRef := u.openRouterClient.ResolveReferences(ctx, history, currentStateStr)
				if errRef != nil {
					log.Printf("[Reference Resolution Warning]: %v", errRef)
				} else {
					resolvedRefs = refs
					resolvedEnts = ents
					refSuccess = true
				}
				return nil
			})
		} else if len(history) > 0 {
			references = state.References
			entities = state.Entities
			if references == nil {
				references = make(map[string]string)
			}
		}

		// Wait for all concurrent LLM calls to finish
		_ = g.Wait()

		// Apply concurrent results safely
		needsSave := false
		if extractSuccess {
			state.Summary = newSummary
			state.ActiveTopic = activeTopic
			state.State = json.RawMessage(extractedState)
			needsSave = true
		}
		if refSuccess {
			references = resolvedRefs
			entities = resolvedEnts
			state.References = references
			state.Entities = entities
			needsSave = true
		}

		if needsSave {
			_ = u.roomRepo.UpsertState(ctx, state)
		}

	} else {
		log.Printf("[Context Engine]: FastMode enabled. Bypassing synchronous LLM calls.")
		if len(history) > 0 {
			references = state.References
			entities = state.Entities
			if references == nil {
				references = make(map[string]string)
			}
		}
	}

	var parsedState any
	if err := json.Unmarshal(state.State, &parsedState); err != nil {
		parsedState = make(map[string]any)
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

	// Async Context Evaluation based on sample rate
	if u.cfg.EvaluationSampleRate > 0 && rand.Float64() <= u.cfg.EvaluationSampleRate {
		select {
		case u.evalQueue <- evalTask{Query: req.Query, Pkg: pkg}:
			// Successfully queued
		default:
			log.Printf("[Context Usecase Warning]: eval queue is full, skipping evaluation")
		}
	}

	return pkg, nil
}
