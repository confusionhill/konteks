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

func (u *roomUsecase) CreateRoom(ctx context.Context, tenantID string, externalUserID string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy) (*domain.Room, error) {
	return u.roomRepo.Create(ctx, tenantID, externalUserID, stateSchema, defaultPolicy)
}

func (u *roomUsecase) GetRoomState(ctx context.Context, tenantID string, roomID string) (*domain.RoomState, error) {
	// Verify room exists and belongs to authorized tenant
	room, err := u.roomRepo.GetByID(ctx, tenantID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}

	return u.roomRepo.GetState(ctx, roomID)
}

func (u *roomUsecase) DeleteRoom(ctx context.Context, tenantID string, roomID string) error {
	return u.roomRepo.Delete(ctx, tenantID, roomID)
}

func (u *roomUsecase) ListRoomsOverview(ctx context.Context, tenantID string, search string, sortBy string, sortOrder string, minMessages int, limit int, offset int) ([]domain.RoomOverview, int, error) {
	return u.roomRepo.ListWithOverview(ctx, tenantID, search, sortBy, sortOrder, minMessages, limit, offset)
}
