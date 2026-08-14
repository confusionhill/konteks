package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/dika/contextkit/backend/internal/config"
	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/dika/contextkit/backend/internal/service"
	"github.com/google/uuid"
)

type messageTask struct {
	ProjectID string
	RoomID    string
	Content   string
	Sequence  int64
}

type messageUsecase struct {
	roomRepo         domain.RoomRepository
	messageRepo      domain.MessageRepository
	memoryRepo       domain.MemoryRepository
	openRouterClient domain.OpenRouterServiceClient
	storageProvider  domain.StorageProvider
	cryptoService    service.CryptoService
	cfg              *config.Config
	taskQueue        chan messageTask
}

func NewMessageUsecase(
	roomRepo domain.RoomRepository,
	messageRepo domain.MessageRepository,
	memoryRepo domain.MemoryRepository,
	openRouterClient domain.OpenRouterServiceClient,
	storageProvider domain.StorageProvider,
	cryptoService service.CryptoService,
	cfg *config.Config,
) domain.MessageUsecase {
	mu := &messageUsecase{
		roomRepo:         roomRepo,
		messageRepo:      messageRepo,
		memoryRepo:       memoryRepo,
		openRouterClient: openRouterClient,
		storageProvider:  storageProvider,
		cryptoService:    cryptoService,
		cfg:              cfg,
		taskQueue:        make(chan messageTask, 5000),
	}

	for i := 0; i < 5; i++ {
		go mu.worker()
	}

	return mu
}

func (u *messageUsecase) worker() {
	for task := range u.taskQueue {
		u.processBackgroundUpdate(task)
	}
}

func (u *messageUsecase) processBackgroundUpdate(task messageTask) {
	ctx := context.Background()

	room, err := u.roomRepo.GetByID(ctx, task.ProjectID, task.RoomID)
	if err != nil || room == nil {
		log.Printf("[Background Worker Error]: failed to retrieve room %s: %v", task.RoomID, err)
		return
	}

	log.Printf("[Background Worker]: generating embedding for room %s...", task.RoomID)
	embedding, err := u.openRouterClient.GenerateEmbedding(ctx, task.Content)
	if err != nil {
		log.Printf("[Background Worker Embedding Error]: failed to generate embedding: %v", err)
	} else {
		memory := &domain.Memory{
			ProjectID: task.ProjectID,
			RoomID:    &task.RoomID,
			Content:   task.Content,
			Embedding: embedding,
			Metadata:  "{}",
		}
		if room.ExternalUserID != "" {
			memory.ExternalUserID = &room.ExternalUserID
		}
		if err := u.memoryRepo.Create(ctx, memory); err != nil {
			log.Printf("[Background Worker Memory Save Error]: failed to save memory: %v", err)
		} else {
			log.Printf("[Background Worker]: successfully saved semantic memory chunk.")
		}
	}

	state, err := u.roomRepo.GetState(ctx, task.RoomID)
	if err != nil {
		log.Printf("[Background Worker State Fetch Error]: failed to fetch state: %v", err)
		return
	}
	if state == nil {
		state = &domain.RoomState{RoomID: task.RoomID}
	}

	history, err := u.messageRepo.ListByRoom(ctx, task.RoomID, 15, 0)
	if err == nil && len(history) > 0 {
		decryptedHistory := make([]domain.Message, len(history))
		var key []byte
		if u.cryptoService != nil && u.cfg != nil {
			key = u.cryptoService.ResolveKey("", u.cfg.EncryptionMasterKey)
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

		log.Printf("[Background Worker]: extracting topic, summary, and state variables...")
		newSummary, activeTopic, stateJSON, err := u.openRouterClient.ExtractStateAndSummary(ctx, decryptedHistory, state.Summary, string(room.StateSchema))
		if err != nil {
			log.Printf("[Background Worker State Extraction Error]: failed to extract state: %v", err)
		} else {
			state.Summary = newSummary
			state.ActiveTopic = activeTopic
			if stateJSON != "" {
				state.State = json.RawMessage(stateJSON)
			}
			state.LastSummarizedSeq = task.Sequence

			if err := u.roomRepo.UpsertState(ctx, state); err != nil {
				log.Printf("[Background Worker State Save Error]: failed to save state: %v", err)
			} else {
				log.Printf("[Background Worker]: room state successfully updated. (Topic: '%s', Sequence: %d)", activeTopic, task.Sequence)
			}
		}
	}
}

func (u *messageUsecase) CreateMessage(ctx context.Context, projectID string, roomID string, role string, content string, isEncrypted bool, encryptionMetadata *string) (*domain.Message, error) {
	room, err := u.roomRepo.GetByID(ctx, projectID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	if room.Status == domain.RoomStatusArchived {
		log.Printf("[Message Usecase]: Room %s is archived. Starting Background Hydration...", roomID)
		_ = u.roomRepo.UpdateStatus(ctx, roomID, domain.RoomStatusHydrating)

		go func() {
			bgCtx := context.Background()
			if room.ArchiveS3Path == nil || *room.ArchiveS3Path == "" {
				log.Printf("[Message Usecase Background Error]: room is archived but archive S3 path is empty for room %s", roomID)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
				return
			}

			archiveBytes, err := u.storageProvider.GetObject(bgCtx, *room.ArchiveS3Path)
			if err != nil {
				log.Printf("[Pre-Warming Error] failed to retrieve S3 object: %v", err)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
				return
			}

			decompressed, err := decompressGzip(archiveBytes)
			if err != nil {
				log.Printf("[Pre-Warming Error] failed to decompress gzip: %v", err)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
				return
			}

			var payload ArchivedRoomPayload
			if err := json.Unmarshal(decompressed, &payload); err != nil {
				log.Printf("[Pre-Warming Error] failed to parse JSON: %v", err)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
				return
			}

			// We only need to hydrate older messages since the newest 3 and room state are already in the DB
			var olderMessages []domain.Message
			if len(payload.Messages) > 3 {
				olderMessages = payload.Messages[:len(payload.Messages)-3]
			}

			log.Printf("[Pre-Warming]: Spawning Stage 2 Backfill for room %s...", roomID)
			errHydrate := u.roomRepo.HydrateRoomTx(bgCtx, roomID, nil, olderMessages, payload.Memories)
			if errHydrate != nil {
				log.Printf("[Pre-Warming Error] Stage 2 Backfill failed: %v", errHydrate)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
			} else {
				log.Printf("[Pre-Warming]: Stage 2 Backfill completed successfully for room %s.", roomID)
			}
		}()
	}

	msg := &domain.Message{
		ID:                 uuid.New().String(),
		RoomID:             roomID,
		Role:               role,
		Content:            content,
		IsEncrypted:        isEncrypted,
		EncryptionMetadata: encryptionMetadata,
	}
	if err := u.messageRepo.Create(ctx, msg); err != nil {
		return nil, err
	}

	select {
	case u.taskQueue <- messageTask{
		ProjectID: projectID,
		RoomID:    roomID,
		Content:   content,
		Sequence:  msg.Sequence,
	}:
		log.Printf("[API]: Message persisted. Queued state update task for room %s (Sequence: %d).", roomID, msg.Sequence)
	default:
		log.Printf("[API Warning]: Background task queue full. Dropped update task for room %s.", roomID)
	}

	return msg, nil
}

func (u *messageUsecase) ListMessages(ctx context.Context, projectID string, roomID string, limit int, beforeSeq int64) ([]domain.Message, error) {
	room, err := u.roomRepo.GetByID(ctx, projectID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	return u.messageRepo.ListByRoom(ctx, roomID, limit, beforeSeq)
}

func (u *messageUsecase) ListProjectLogs(ctx context.Context, projectID string, roomID string, externalUserID string, search string, role string, sortBy string, sortOrder string, limit int, offset int) ([]domain.ProjectLogEntry, int, error) {
	return u.messageRepo.ListProjectLogs(ctx, projectID, roomID, externalUserID, search, role, sortBy, sortOrder, limit, offset)
}
