package service

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dika/contextkit/backend/internal/domain"
)

// roomStore is the room persistence contract used by MCPRoomService. It is
// satisfied by repository.MCPRoomRepository.
type roomStore interface {
	Create(ctx context.Context, projectID string, externalUserID string, description string, parentRoomID *string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy, initialState json.RawMessage) (*domain.Room, error)
	GetByID(ctx context.Context, projectID string, roomID string) (*domain.Room, error)
	Update(ctx context.Context, projectID string, roomID string, description *string, parentRoomID *string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy) (*domain.Room, error)
	UpdateStatus(ctx context.Context, roomID string, status string) error
	ListByProject(ctx context.Context, projectID string, externalUserID string, limit int, offset int) ([]domain.RoomOverview, int, error)
	ListChildRooms(ctx context.Context, projectID string, parentRoomID string) ([]domain.ChildAgentOverview, error)
}

// stateStore is the room-state persistence contract for MCP services.
type stateStore interface {
	Get(ctx context.Context, roomID string) (*domain.RoomState, error)
	Patch(ctx context.Context, roomID string, updates map[string]any, deletes []string) (*domain.RoomState, error)
}

// MCPRoomService orchestrates project-scoped room lifecycle operations for the
// MCP server. All queries are scoped to the auth context's project_id.
type MCPRoomService struct {
	roomRepo  roomStore
	stateRepo stateStore
}

func NewMCPRoomService(roomRepo roomStore, stateRepo stateStore) *MCPRoomService {
	return &MCPRoomService{roomRepo: roomRepo, stateRepo: stateRepo}
}

func (s *MCPRoomService) CreateRoom(ctx context.Context, auth domain.AuthContext, externalUserID string, description string, parentRoomID *string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy, initialState json.RawMessage) (*domain.Room, error) {
	return s.roomRepo.Create(ctx, auth.ProjectID, externalUserID, description, parentRoomID, stateSchema, defaultPolicy, initialState)
}

// GetRoom returns the room and its state, or an error if not found in scope.
func (s *MCPRoomService) GetRoom(ctx context.Context, auth domain.AuthContext, roomID string) (*domain.Room, *domain.RoomState, error) {
	room, err := s.roomRepo.GetByID(ctx, auth.ProjectID, roomID)
	if err != nil {
		return nil, nil, err
	}
	if room == nil {
		return nil, nil, fmt.Errorf("room not found or unauthorized")
	}
	state, err := s.stateRepo.Get(ctx, roomID)
	if err != nil {
		return nil, nil, err
	}
	if state == nil {
		state = &domain.RoomState{
			RoomID:      roomID,
			Summary:     "",
			ActiveTopic: "",
			State:       json.RawMessage("{}"),
		}
	}
	return room, state, nil
}

func (s *MCPRoomService) GetRoomState(ctx context.Context, auth domain.AuthContext, roomID string) (*domain.RoomState, error) {
	room, err := s.roomRepo.GetByID(ctx, auth.ProjectID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}
	state, err := s.stateRepo.Get(ctx, roomID)
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
	return state, nil
}

func (s *MCPRoomService) ListRooms(ctx context.Context, auth domain.AuthContext, externalUserID string, limit int, offset int) ([]domain.RoomOverview, int, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	return s.roomRepo.ListByProject(ctx, auth.ProjectID, externalUserID, limit, offset)
}

func (s *MCPRoomService) UpdateRoomState(ctx context.Context, auth domain.AuthContext, roomID string, updates map[string]any, deletes []string) (*domain.RoomState, error) {
	room, err := s.roomRepo.GetByID(ctx, auth.ProjectID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}
	return s.stateRepo.Patch(ctx, roomID, updates, deletes)
}

func (s *MCPRoomService) UpdateRoom(ctx context.Context, auth domain.AuthContext, roomID string, description *string, parentRoomID *string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy) (*domain.Room, error) {
	room, err := s.roomRepo.Update(ctx, auth.ProjectID, roomID, description, parentRoomID, stateSchema, defaultPolicy)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}
	return room, nil
}

func (s *MCPRoomService) ArchiveRoom(ctx context.Context, auth domain.AuthContext, roomID string) (*domain.Room, error) {
	room, err := s.roomRepo.GetByID(ctx, auth.ProjectID, roomID)
	if err != nil {
		return nil, err
	}
	if room == nil {
		return nil, fmt.Errorf("room not found or unauthorized")
	}
	if room.Status == domain.RoomStatusArchived {
		return room, nil
	}
	if err := s.roomRepo.UpdateStatus(ctx, roomID, domain.RoomStatusArchived); err != nil {
		return nil, err
	}
	room.Status = domain.RoomStatusArchived
	return room, nil
}

func (s *MCPRoomService) ListChildRooms(ctx context.Context, auth domain.AuthContext, parentRoomID string) ([]domain.ChildAgentOverview, error) {
	return s.roomRepo.ListChildRooms(ctx, auth.ProjectID, parentRoomID)
}