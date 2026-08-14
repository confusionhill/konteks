package usecase

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dika/contextkit/backend/internal/domain"
)

type roomUsecase struct {
	roomRepo domain.RoomRepository
}

func NewRoomUsecase(roomRepo domain.RoomRepository) domain.RoomUsecase {
	return &roomUsecase{roomRepo: roomRepo}
}

func (u *roomUsecase) CreateRoom(ctx context.Context, projectID string, externalUserID string, description string, parentRoomID *string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy) (*domain.Room, error) {
	return u.roomRepo.Create(ctx, projectID, externalUserID, description, parentRoomID, stateSchema, defaultPolicy)
}

func (u *roomUsecase) GetRoomState(ctx context.Context, projectID string, roomID string) (*domain.RoomState, error) {
	room, err := u.roomRepo.GetByID(ctx, projectID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	return u.roomRepo.GetState(ctx, roomID)
}

func (u *roomUsecase) DeleteRoom(ctx context.Context, projectID string, roomID string) error {
	return u.roomRepo.Delete(ctx, projectID, roomID)
}

func (u *roomUsecase) ListRoomsOverview(ctx context.Context, projectID string, search string, status string, sortBy string, sortOrder string, minMessages int, limit int, offset int) ([]domain.RoomOverview, int, error) {
	return u.roomRepo.ListWithOverview(ctx, projectID, search, status, sortBy, sortOrder, minMessages, limit, offset)
}

func (u *roomUsecase) ListChildRooms(ctx context.Context, projectID string, parentRoomID string) ([]domain.ChildAgentOverview, error) {
	parentRoom, err := u.roomRepo.GetByID(ctx, projectID, parentRoomID)
	if err != nil {
		return nil, err
	}
	if parentRoom == nil {
		return nil, fmt.Errorf("parent room not found or unauthorized")
	}

	return u.roomRepo.ListChildRooms(ctx, projectID, parentRoomID)
}

func (u *roomUsecase) UpdateRoom(ctx context.Context, projectID string, roomID string, description *string, parentRoomID *string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy) (*domain.Room, error) {
	return u.roomRepo.Update(ctx, projectID, roomID, description, parentRoomID, stateSchema, defaultPolicy)
}
