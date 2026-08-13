package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dika/contextkit/backend/internal/domain"
)

type eventUsecase struct {
	roomRepo  domain.RoomRepository
	eventRepo domain.EventRepository
}

func NewEventUsecase(roomRepo domain.RoomRepository, eventRepo domain.EventRepository) domain.EventUsecase {
	return &eventUsecase{
		roomRepo:  roomRepo,
		eventRepo: eventRepo,
	}
}

func (u *eventUsecase) LogEvent(ctx context.Context, projectID string, roomID string, eventType string, payload any) (*domain.RoomEvent, error) {
	room, err := u.roomRepo.GetByID(ctx, projectID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	event := &domain.RoomEvent{
		RoomID:    roomID,
		EventType: eventType,
		Payload:   string(payloadBytes),
	}

	if err := u.eventRepo.Create(ctx, event); err != nil {
		return nil, err
	}

	return event, nil
}

func (u *eventUsecase) ListEvents(ctx context.Context, projectID string, roomID string, limit int) ([]domain.RoomEvent, error) {
	room, err := u.roomRepo.GetByID(ctx, projectID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	return u.eventRepo.ListByRoom(ctx, roomID, limit)
}
