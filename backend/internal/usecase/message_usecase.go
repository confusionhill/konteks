package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/dika/contextkit/backend/internal/domain"
)

type messageTask struct {
	TenantID string
	RoomID   string
	Content  string
}

type messageUsecase struct {
	roomRepo         domain.RoomRepository
	messageRepo      domain.MessageRepository
	memoryRepo       domain.MemoryRepository
	openRouterClient domain.OpenRouterServiceClient
	taskQueue        chan messageTask
}

func NewMessageUsecase(
	roomRepo domain.RoomRepository,
	messageRepo domain.MessageRepository,
	memoryRepo domain.MemoryRepository,
	openRouterClient domain.OpenRouterServiceClient,
) domain.MessageUsecase {
	mu := &messageUsecase{
		roomRepo:         roomRepo,
		messageRepo:      messageRepo,
		memoryRepo:       memoryRepo,
		openRouterClient: openRouterClient,
		taskQueue:        make(chan messageTask, 5000),
	}

	// Start 5 background worker goroutines
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

	// 1. Fetch room
	room, err := u.roomRepo.GetByID(ctx, task.TenantID, task.RoomID)
	if err != nil || room == nil {
		log.Printf("[Background Worker Error]: failed to retrieve room %s: %v", task.RoomID, err)
		return
	}

	// 2. Generate embedding vector for the message content
	log.Printf("[Background Worker]: generating embedding for room %s...", task.RoomID)
	embedding, err := u.openRouterClient.GenerateEmbedding(ctx, task.Content)
	if err != nil {
		log.Printf("[Background Worker Embedding Error]: failed to generate embedding: %v", err)
	} else {
		// 3. Save embedding as a semantic Memory chunk
		memory := &domain.Memory{
			TenantID:  task.TenantID,
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

	// 4. Retrieve current room state and last 15 messages for LLM extraction
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
		// 5. Synchronously extract summary, active topic, and state payload
		log.Printf("[Background Worker]: extracting topic, summary, and state variables...")
		newSummary, activeTopic, stateJSON, err := u.openRouterClient.ExtractStateAndSummary(ctx, history, state.Summary, string(room.StateSchema))
		if err != nil {
			log.Printf("[Background Worker State Extraction Error]: failed to extract state: %v", err)
		} else {
			state.Summary = newSummary
			state.ActiveTopic = activeTopic
			state.State = json.RawMessage(stateJSON)

			// 6. Upsert state
			if err := u.roomRepo.UpsertState(ctx, state); err != nil {
				log.Printf("[Background Worker State Save Error]: failed to save state: %v", err)
			} else {
				log.Printf("[Background Worker]: room state successfully updated. (Topic: '%s')", activeTopic)
			}
		}
	}
}

func (u *messageUsecase) CreateMessage(ctx context.Context, tenantID string, roomID string, role string, content string) (*domain.Message, error) {
	// 1. Verify room ownership
	room, err := u.roomRepo.GetByID(ctx, tenantID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	// 2. Persist raw message
	msg := &domain.Message{
		RoomID:  roomID,
		Role:    role,
		Content: content,
	}
	if err := u.messageRepo.Create(ctx, msg); err != nil {
		return nil, err
	}

	// 3. Queue task for background processing
	select {
	case u.taskQueue <- messageTask{
		TenantID: tenantID,
		RoomID:   roomID,
		Content:  content,
	}:
		log.Printf("[API]: Message persisted. Queued state update task for room %s.", roomID)
	default:
		log.Printf("[API Warning]: Background task queue full. Dropped update task for room %s.", roomID)
	}

	return msg, nil
}

func (u *messageUsecase) ListMessages(ctx context.Context, tenantID string, roomID string, limit int, beforeSeq int64) ([]domain.Message, error) {
	// Verify room ownership
	room, err := u.roomRepo.GetByID(ctx, tenantID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	return u.messageRepo.ListByRoom(ctx, roomID, limit, beforeSeq)
}
