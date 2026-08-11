package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/dika/contextkit/backend/internal/domain"
)

type messageUsecase struct {
	roomRepo         domain.RoomRepository
	messageRepo      domain.MessageRepository
	memoryRepo       domain.MemoryRepository
	openRouterClient domain.OpenRouterServiceClient
}

func NewMessageUsecase(
	roomRepo domain.RoomRepository,
	messageRepo domain.MessageRepository,
	memoryRepo domain.MemoryRepository,
	openRouterClient domain.OpenRouterServiceClient,
) domain.MessageUsecase {
	return &messageUsecase{
		roomRepo:         roomRepo,
		messageRepo:      messageRepo,
		memoryRepo:       memoryRepo,
		openRouterClient: openRouterClient,
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

	// 3. Generate embedding vector for the message content
	embedding, err := u.openRouterClient.GenerateEmbedding(ctx, content)
	if err != nil {
		log.Printf("[Embedding Error]: failed to generate embedding: %v", err)
	} else {
		// 4. Save embedding as a semantic Memory chunk
		memory := &domain.Memory{
			TenantID:       tenantID,
			RoomID:         &roomID,
			Content:        content,
			Embedding:      embedding,
			Metadata:       "{}",
		}
		if room.ExternalUserID != "" {
			memory.ExternalUserID = &room.ExternalUserID
		}
		if err := u.memoryRepo.Create(ctx, memory); err != nil {
			log.Printf("[Memory Save Error]: failed to save memory: %v", err)
		}
	}

	// 5. Retrieve current room state and last 15 messages for LLM extraction
	state, err := u.roomRepo.GetState(ctx, roomID)
	if err != nil {
		return msg, nil // Return message even if state updates fail (robustness)
	}
	if state == nil {
		state = &domain.RoomState{RoomID: roomID}
	}

	history, err := u.messageRepo.ListByRoom(ctx, roomID, 15, 0)
	if err == nil && len(history) > 0 {
		// 6. Synchronously extract summary, active topic, and state payload
		newSummary, activeTopic, stateJSON, err := u.openRouterClient.ExtractStateAndSummary(ctx, history, state.Summary)
		if err != nil {
			log.Printf("[State Extraction Error]: failed to extract state: %v", err)
		} else {
			state.Summary = newSummary
			state.ActiveTopic = activeTopic
			state.State = json.RawMessage(stateJSON)

			// 7. Upsert state
			if err := u.roomRepo.UpsertState(ctx, state); err != nil {
				log.Printf("[State Save Error]: failed to save state: %v", err)
			}
		}
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
