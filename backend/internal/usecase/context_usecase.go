package usecase

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"

	"github.com/dika/contextkit/backend/internal/config"
	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/dika/contextkit/backend/internal/service"
)

type contextUsecase struct {
	roomRepo         domain.RoomRepository
	messageRepo      domain.MessageRepository
	memoryRepo       domain.MemoryRepository
	openRouterClient domain.OpenRouterServiceClient
	storageProvider  domain.StorageProvider
	evalRepo         domain.EvaluationRepository
	cryptoService    service.CryptoService
	cfg              *config.Config
	evalQueue        chan evalTask
}

type evalTask struct {
	ProjectID          string
	Query              string
	Pkg                *domain.ContextPackage
	IsEncrypted        bool
	EncryptionMetadata *string
	RequestKey         string
}

func NewContextUsecase(
	roomRepo domain.RoomRepository,
	messageRepo domain.MessageRepository,
	memoryRepo domain.MemoryRepository,
	openRouterClient domain.OpenRouterServiceClient,
	storageProvider domain.StorageProvider,
	evalRepo domain.EvaluationRepository,
	cryptoService service.CryptoService,
	cfg *config.Config,
) domain.ContextUsecase {
	u := &contextUsecase{
		roomRepo:         roomRepo,
		messageRepo:      messageRepo,
		memoryRepo:       memoryRepo,
		openRouterClient: openRouterClient,
		storageProvider:  storageProvider,
		evalRepo:         evalRepo,
		cryptoService:    cryptoService,
		cfg:              cfg,
		evalQueue:        make(chan evalTask, 1000),
	}

	for i := 0; i < 2; i++ {
		go u.evaluationWorker()
	}

	return u
}

func (u *contextUsecase) evaluationWorker() {
	for task := range u.evalQueue {
		if task.IsEncrypted {
			log.Printf("[Evaluation Worker]: Skipping context evaluation for encrypted payload (Room %s)", task.Pkg.RoomID)
			continue
		}
		ctx := context.Background()
		evalQuery := task.Query
		var key []byte
		if u.cryptoService != nil && u.cfg != nil {
			key = u.cryptoService.ResolveKey(task.RequestKey, u.cfg.EncryptionMasterKey)
		}

		if task.IsEncrypted && len(key) > 0 {
			var nonce string
			if task.EncryptionMetadata != nil {
				var meta map[string]interface{}
				if json.Unmarshal([]byte(*task.EncryptionMetadata), &meta) == nil {
					if n, ok := meta["nonce"].(string); ok {
						nonce = n
					}
				}
			}
			if decrypted, errDec := u.cryptoService.Decrypt(task.Query, nonce, key); errDec == nil {
				evalQuery = decrypted
			}
		}

		evalPkg := *task.Pkg
		if len(evalPkg.RelevantMessages) > 0 && len(key) > 0 {
			decryptedMsgs := make([]domain.Message, len(evalPkg.RelevantMessages))
			for i, m := range evalPkg.RelevantMessages {
				decryptedMsgs[i] = m
				if m.IsEncrypted {
					var nonce string
					if m.EncryptionMetadata != nil {
						var meta map[string]interface{}
						if json.Unmarshal([]byte(*m.EncryptionMetadata), &meta) == nil {
							if n, ok := meta["nonce"].(string); ok {
								nonce = n
							}
						}
					}
					if decrypted, errDec := u.cryptoService.Decrypt(m.Content, nonce, key); errDec == nil {
						decryptedMsgs[i].Content = decrypted
					}
				}
			}
			evalPkg.RelevantMessages = decryptedMsgs
		}

		eval, err := u.openRouterClient.EvaluateContext(ctx, evalQuery, &evalPkg)
		if err != nil {
			log.Printf("[Evaluation Worker Error]: failed to evaluate context: %v", err)
			continue
		}

		eval.ID = uuid.NewString()
		eval.ProjectID = task.ProjectID
		eval.RoomID = task.Pkg.RoomID
		eval.IsEncrypted = task.IsEncrypted
		eval.EncryptionMetadata = task.EncryptionMetadata

		// Preserve original encrypted ContextSnapshot in database
		if task.IsEncrypted {
			cleanedOriginalPkg := *task.Pkg
			if len(cleanedOriginalPkg.Memories) > 0 {
				cleanedMem := make([]domain.Memory, len(cleanedOriginalPkg.Memories))
				for i, m := range cleanedOriginalPkg.Memories {
					c := m
					c.Embedding = nil
					cleanedMem[i] = c
				}
				cleanedOriginalPkg.Memories = cleanedMem
			}
			if snapBytes, errSnap := json.Marshal(cleanedOriginalPkg); errSnap == nil {
				eval.ContextSnapshot = snapBytes
			}
		}

		if task.IsEncrypted && len(key) > 0 && u.cryptoService != nil {
			encQ, nonce, errEncQ := u.cryptoService.Encrypt(eval.Query, key)
			encR, _, errEncR := u.cryptoService.Encrypt(eval.Reasoning, key)
			if errEncQ == nil && errEncR == nil {
				eval.Query = encQ
				eval.Reasoning = encR
				metaJSON, _ := json.Marshal(map[string]string{
					"alg":         "AES-256-GCM",
					"key_version": "v1",
					"nonce":       nonce,
				})
				metaStr := string(metaJSON)
				eval.EncryptionMetadata = &metaStr
			}
		}

		if err := u.evalRepo.Create(ctx, eval); err != nil {
			log.Printf("[Evaluation Worker Error]: failed to save evaluation: %v", err)
		} else {
			log.Printf("[Evaluation Worker]: successfully evaluated context for room %s (Score: %.2f/%.2f, Encrypted: %t)", eval.RoomID, eval.RelevanceScore, eval.CompletenessScore, eval.IsEncrypted)
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

func classifyStrategy(query string) string {
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

func hasPronouns(query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return false
	}

	greetings := map[string]bool{
		"hi": true, "hello": true, "hey": true, "yo": true, "greetings": true,
		"thanks": true, "thank you": true, "ok": true, "okay": true, "yes": true, "no": true,
		"help": true, "info": true, "start": true, "stop": true,
	}
	if greetings[q] {
		return false
	}

	pronouns := []string{" it ", " that ", " which ", " this ", " them ", " option ", " same ", " the hotel ", " the flight ", " the room "}
	padded := " " + q + " "
	for _, p := range pronouns {
		if strings.Contains(padded, p) {
			return true
		}
	}
	return false
}

func (u *contextUsecase) GetContextPackage(ctx context.Context, projectID string, roomID string, req domain.GetContextRequest) (*domain.ContextPackage, error) {
	room, err := u.roomRepo.GetByID(ctx, projectID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	if room.Status == domain.RoomStatusArchived {
		log.Printf("[Context Engine]: Room %s is archived. Triggering Background Hydration...", roomID)
		err = u.roomRepo.UpdateStatus(ctx, roomID, domain.RoomStatusHydrating)
		if err != nil {
			return nil, fmt.Errorf("failed to lock room status to hydrating: %w", err)
		}

		go func() {
			bgCtx := context.Background()

			if room.ArchiveS3Path == nil || *room.ArchiveS3Path == "" {
				log.Printf("[Context Engine Background Error]: room is archived but archive S3 path is empty for room %s", roomID)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
				return
			}

			archiveBytes, err := u.storageProvider.GetObject(bgCtx, *room.ArchiveS3Path)
			if err != nil {
				log.Printf("[Context Engine Background Error]: failed to retrieve room archive: %v", err)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
				return
			}

			decompressed, err := decompressGzip(archiveBytes)
			if err != nil {
				log.Printf("[Context Engine Background Error]: failed to decompress archive data: %v", err)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
				return
			}

			var payload ArchivedRoomPayload
			if err := json.Unmarshal(decompressed, &payload); err != nil {
				log.Printf("[Context Engine Background Error]: failed to parse archive JSON: %v", err)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
				return
			}

			log.Printf("[Context Engine]: Spawning Stage 2 Backfill for room %s...", roomID)

			// We only need to hydrate older messages since the newest 3 and room state are already in the DB
			var olderMessages []domain.Message
			if len(payload.Messages) > 3 {
				olderMessages = payload.Messages[:len(payload.Messages)-3]
			}

			errHydrate := u.roomRepo.HydrateRoomTx(bgCtx, roomID, nil, olderMessages, payload.Memories)
			if errHydrate != nil {
				log.Printf("[Context Engine Background Hydration Error]: %v", errHydrate)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
			} else {
				log.Printf("[Context Engine]: Stage 2 Backfill completed successfully for room %s.", roomID)
			}
		}()

		// We do not block here. We continue processing the request using the hot cache from DB.
		room.Status = domain.RoomStatusActive
	}

	if room.Status == domain.RoomStatusHydrating {
		log.Printf("[Context Engine]: Room %s is currently hydrating. Serving from hot cache...", roomID)
		// We do not block here. We continue processing the request using the hot cache from DB.
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

	if msgLimit > policy.MaxHistoryDepth {
		msgLimit = policy.MaxHistoryDepth
	}

	// Parallel DB fetch: GetState and GetMaxSequence have no dependency on each other
	var state *domain.RoomState
	var maxSeq int64
	{
		var dbGroup errgroup.Group
		dbGroup.Go(func() error {
			s, errState := u.roomRepo.GetState(ctx, roomID)
			if errState != nil {
				return errState
			}
			state = s
			return nil
		})
		dbGroup.Go(func() error {
			seq, errSeq := u.messageRepo.GetMaxSequence(ctx, roomID)
			if errSeq != nil {
				seq = 0
			}
			maxSeq = seq
			return nil
		})
		if err := dbGroup.Wait(); err != nil {
			return nil, err
		}
	}
	if state == nil {
		state = &domain.RoomState{
			RoomID:      roomID,
			Summary:     "",
			ActiveTopic: "",
			State:       json.RawMessage("{}"),
		}
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

	decryptedHistory := make([]domain.Message, len(history))
	var key []byte
	if u.cryptoService != nil && u.cfg != nil {
		key = u.cryptoService.ResolveKey(req.RequestKey, u.cfg.EncryptionMasterKey)
	}
	for i, m := range history {
		decryptedHistory[i] = m
		if m.IsEncrypted && len(key) > 0 {
			var nonce string
			if m.EncryptionMetadata != nil {
				var meta map[string]interface{}
				if json.Unmarshal([]byte(*m.EncryptionMetadata), &meta) == nil {
					if n, ok := meta["nonce"].(string); ok {
						nonce = n
					}
				}
			}
			if decrypted, errDec := u.cryptoService.Decrypt(m.Content, nonce, key); errDec == nil {
				decryptedHistory[i].Content = decrypted
			}
		}
	}

	var matchedMemories []domain.Memory
	references := make(map[string]string)
	entities := []domain.Entity{}

	if !policy.FastMode {
		var g errgroup.Group

		var extractSuccess bool
		var newSummary, activeTopic string
		var extractedState []byte

		if len(req.DynamicSchema) > 0 && string(req.DynamicSchema) != "null" && string(req.DynamicSchema) != "{}" {
			g.Go(func() error {
				log.Printf("[Context Engine]: dynamic schema provided. Performing on-the-fly state extraction...")
				sum, topic, extState, _, errExtract := u.openRouterClient.ExtractStateAndSummary(ctx, decryptedHistory, state.Summary, string(req.DynamicSchema))
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

		var refSuccess bool
		var resolvedRefs map[string]string
		var resolvedEnts []domain.Entity

		if len(decryptedHistory) > 0 && hasPronouns(req.Query) {
			currentStateStr := string(state.State)
			g.Go(func() error {
				log.Printf("[Context Engine]: Pronoun detected in query. Resolving references...")
				refs, ents, errRef := u.openRouterClient.ResolveReferences(ctx, decryptedHistory, currentStateStr)
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

		_ = g.Wait()

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
		children, errChild := u.roomRepo.ListChildRooms(ctx, projectID, roomID)
		if errChild == nil && len(children) > 0 {
			childAgents = children
		} else {
			childAgents = []domain.ChildAgentOverview{}
		}
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
		ChildAgents:      childAgents,
		StrategyUsed:     strategy,
		TokensEstimated:  estimatedTokens,
		TokensProcessed:  processedTokens,
	}

	if !req.IsEncrypted && u.cfg.EvaluationSampleRate > 0 && rand.Float64() <= u.cfg.EvaluationSampleRate {
		select {
		case u.evalQueue <- evalTask{
			ProjectID:          projectID,
			Query:              req.Query,
			Pkg:                pkg,
			IsEncrypted:        req.IsEncrypted,
			EncryptionMetadata: req.EncryptionMetadata,
			RequestKey:         req.RequestKey,
		}:
		default:
			log.Printf("[Context Usecase Warning]: eval queue is full, skipping evaluation")
		}
	}

	return pkg, nil
}

func (u *contextUsecase) Warmup(ctx context.Context, projectID string, roomID string) error {
	room, err := u.roomRepo.GetByID(ctx, projectID, roomID)
	if err != nil {
		return err
	}
	if room == nil {
		return fmt.Errorf("room not found or unauthorized")
	}

	if room.Status == domain.RoomStatusArchived {
		log.Printf("[Context Engine]: Proactive Warmup triggered for archived room %s. Starting Background Hydration...", roomID)
		err = u.roomRepo.UpdateStatus(ctx, roomID, domain.RoomStatusHydrating)
		if err != nil {
			return fmt.Errorf("failed to lock room status to hydrating: %w", err)
		}

		go func() {
			bgCtx := context.Background()

			if room.ArchiveS3Path == nil || *room.ArchiveS3Path == "" {
				log.Printf("[Context Engine Background Error]: room is archived but archive S3 path is empty for room %s", roomID)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
				return
			}

			archiveBytes, err := u.storageProvider.GetObject(bgCtx, *room.ArchiveS3Path)
			if err != nil {
				log.Printf("[Context Engine Background Error]: failed to retrieve room archive: %v", err)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
				return
			}

			decompressed, err := decompressGzip(archiveBytes)
			if err != nil {
				log.Printf("[Context Engine Background Error]: failed to decompress archive data: %v", err)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
				return
			}

			var payload ArchivedRoomPayload
			if err := json.Unmarshal(decompressed, &payload); err != nil {
				log.Printf("[Context Engine Background Error]: failed to parse archive JSON: %v", err)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
				return
			}

			log.Printf("[Context Engine]: Spawning Stage 2 Backfill for room %s...", roomID)

			var olderMessages []domain.Message
			if len(payload.Messages) > 3 {
				olderMessages = payload.Messages[:len(payload.Messages)-3]
			}

			errHydrate := u.roomRepo.HydrateRoomTx(bgCtx, roomID, nil, olderMessages, payload.Memories)
			if errHydrate != nil {
				log.Printf("[Context Engine Background Hydration Error]: %v", errHydrate)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
			} else {
				log.Printf("[Context Engine]: Stage 2 Backfill completed successfully for room %s.", roomID)
			}
		}()
	}
	return nil
}
