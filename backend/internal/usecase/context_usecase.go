package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

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

func (u *contextUsecase) GetContextPackage(ctx context.Context, tenantID string, roomID string, query string, maxTokens int) (*domain.ContextPackage, error) {
	// 1. Verify room ownership
	room, err := u.roomRepo.GetByID(ctx, tenantID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	// 2. Fetch RoomState
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

	// Unmarshal state json so it's returned as raw generic interface/map
	var parsedState any
	if err := json.Unmarshal(state.State, &parsedState); err != nil {
		parsedState = make(map[string]any)
	}

	// 3. Search semantic memories if query is provided
	var matchedMemories []domain.Memory
	if query != "" {
		embedding, err := u.openRouterClient.GenerateEmbedding(ctx, query)
		if err != nil {
			log.Printf("[Context Query Embedding Error]: failed to generate embedding: %v", err)
		} else {
			var extUserID *string
			if room.ExternalUserID != "" {
				extUserID = &room.ExternalUserID
			}
			var errSearch error
			matchedMemories, errSearch = u.memoryRepo.SearchSemantic(ctx, roomID, extUserID, embedding, 5)
			if errSearch != nil {
				log.Printf("[Context Memory Search Error]: %v", errSearch)
			}
		}
	}

	// 4. Retrieve recent message history (last 10 messages)
	history, err := u.messageRepo.ListByRoom(ctx, roomID, 10, 0)
	if err != nil {
		history = []domain.Message{}
	}

	// 5. Assemble context package
	pkg := &domain.ContextPackage{
		RoomID:           roomID,
		Summary:          state.Summary,
		ActiveTopic:      state.ActiveTopic,
		State:            parsedState,
		RelevantMessages: history,
		Memories:         matchedMemories,
		Entities:         []domain.Entity{}, // Initialize to prevent JSON null
		References:       make(map[string]string),
	}

	return pkg, nil
}
