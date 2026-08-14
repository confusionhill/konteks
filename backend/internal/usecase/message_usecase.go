package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/dika/contextkit/backend/internal/domain"
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
	taskQueue        chan messageTask
}

func NewMessageUsecase(
	roomRepo domain.RoomRepository,
	messageRepo domain.MessageRepository,
	memoryRepo domain.MemoryRepository,
	openRouterClient domain.OpenRouterServiceClient,
	storageProvider domain.StorageProvider,
) domain.MessageUsecase {
	mu := &messageUsecase{
		roomRepo:         roomRepo,
		messageRepo:      messageRepo,
		memoryRepo:       memoryRepo,
		openRouterClient: openRouterClient,
		storageProvider:  storageProvider,
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
		log.Printf("[Background Worker]: extracting topic, summary, and state variables...")
		newSummary, activeTopic, stateJSON, err := u.openRouterClient.ExtractStateAndSummary(ctx, history, state.Summary, string(room.StateSchema))
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
		log.Printf("[Message Usecase]: Room %s is archived. Starting Parallel Pre-Warming...", roomID)
		_ = u.roomRepo.UpdateStatus(ctx, roomID, domain.RoomStatusHydrating)

		go func() {
			bgCtx := context.Background()
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

			var stage1Msgs []domain.Message
			var stage2Msgs []domain.Message
			if len(payload.Messages) > 3 {
				stage1Msgs = payload.Messages[len(payload.Messages)-3:]
				stage2Msgs = payload.Messages[:len(payload.Messages)-3]
			} else {
				stage1Msgs = payload.Messages
			}

			err = u.roomRepo.HydrateRoomTx(bgCtx, roomID, &payload.RoomState, stage1Msgs, nil)
			if err != nil {
				log.Printf("[Pre-Warming Error] Stage 1 Hydration failed: %v", err)
				_ = u.roomRepo.UpdateStatus(bgCtx, roomID, domain.RoomStatusArchived)
				return
			}

			msg := &domain.Message{
				ID:                 uuid.New().String(),
				RoomID:             roomID,
				Role:               role,
				Content:            content,
				IsEncrypted:        isEncrypted,
				EncryptionMetadata: encryptionMetadata,
			}
			if err := u.messageRepo.Create(bgCtx, msg); err != nil {
				log.Printf("[Pre-Warming Error] failed to save message: %v", err)
			} else {
				select {
				case u.taskQueue <- messageTask{
					ProjectID: projectID,
					RoomID:    roomID,
					Content:   content,
					Sequence:  msg.Sequence,
				}:
				default:
				}
			}

			log.Printf("[Pre-Warming]: Stage 1 Hydrate complete. Spawning Stage 2 Backfill for room %s...", roomID)
			_ = u.roomRepo.HydrateRoomTx(bgCtx, roomID, nil, stage2Msgs, payload.Memories)
		}()

		msg := &domain.Message{
			ID:                 uuid.New().String(),
			RoomID:             roomID,
			Role:               role,
			Content:            content,
			IsEncrypted:        isEncrypted,
			EncryptionMetadata: encryptionMetadata,
		}
		return msg, nil
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
