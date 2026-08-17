package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/dika/contextkit/backend/internal/domain"
)

type stubRoomStore struct {
	room *domain.Room
	// createCalled captures the project scope passed to Create.
	createProjectID string
	createdRoom     *domain.Room
	overviews       []domain.RoomOverview
	total           int
	err             error
}

func (s *stubRoomStore) Create(ctx context.Context, projectID string, externalUserID string, description string, parentRoomID *string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy, initialState json.RawMessage) (*domain.Room, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.createProjectID = projectID
	room := &domain.Room{ID: "room-1", ProjectID: projectID, ExternalUserID: externalUserID, Description: description, Status: domain.RoomStatusActive}
	if parentRoomID != nil {
		room.ParentRoomID = parentRoomID
	}
	s.createdRoom = room
	return room, nil
}

func (s *stubRoomStore) GetByID(ctx context.Context, projectID string, roomID string) (*domain.Room, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.room != nil && s.room.ID == roomID {
		return s.room, nil
	}
	return nil, nil
}

func (s *stubRoomStore) Update(ctx context.Context, projectID string, roomID string, description *string, parentRoomID *string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy) (*domain.Room, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.room == nil {
		return nil, nil
	}
	return s.room, nil
}

func (s *stubRoomStore) UpdateStatus(ctx context.Context, roomID string, status string) error {
	if s.err != nil {
		return s.err
	}
	if s.room != nil {
		s.room.Status = status
	}
	return nil
}

func (s *stubRoomStore) ListByProject(ctx context.Context, projectID string, externalUserID string, limit int, offset int) ([]domain.RoomOverview, int, error) {
	if s.err != nil {
		return nil, 0, s.err
	}
	return s.overviews, s.total, nil
}

func (s *stubRoomStore) ListChildRooms(ctx context.Context, projectID string, parentRoomID string) ([]domain.ChildAgentOverview, error) {
	return []domain.ChildAgentOverview{}, nil
}

type stubStateStore struct {
	state *domain.RoomState
	err   error
}

func (s *stubStateStore) Get(ctx context.Context, roomID string) (*domain.RoomState, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.state, nil
}

func (s *stubStateStore) Patch(ctx context.Context, roomID string, updates map[string]any, deletes []string) (*domain.RoomState, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.state == nil {
		s.state = &domain.RoomState{RoomID: roomID, State: json.RawMessage("{}")}
	}
	return s.state, nil
}

func newRoomTestService(room *domain.Room, state *domain.RoomState) (*MCPRoomService, *stubRoomStore, *stubStateStore) {
	rs := &stubRoomStore{room: room}
	ss := &stubStateStore{state: state}
	return NewMCPRoomService(rs, ss), rs, ss
}

func TestRoomService_CreateRoom_AllowsEmptyExternalUser(t *testing.T) {
	svc, rs, _ := newRoomTestService(nil, nil)
	auth := domain.AuthContext{ProjectID: "proj-a"}
	_, err := svc.CreateRoom(context.Background(), auth, "", "desc", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rs.createProjectID != "proj-a" {
		t.Fatalf("expected project scope proj-a, got %q", rs.createProjectID)
	}
}

func TestRoomService_CreateRoom_ScopesToProject(t *testing.T) {
	svc, rs, _ := newRoomTestService(nil, nil)
	auth := domain.AuthContext{ProjectID: "proj-a"}
	parent := "parent-1"
	room, err := svc.CreateRoom(context.Background(), auth, "user-42", "test room", &parent, nil, nil, json.RawMessage(`{"x":1}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rs.createProjectID != "proj-a" {
		t.Fatalf("Create called with project %q, want proj-a", rs.createProjectID)
	}
	if room.ParentRoomID == nil || *room.ParentRoomID != "parent-1" {
		t.Fatalf("parent room not propagated: %+v", room.ParentRoomID)
	}
}

func TestRoomService_GetRoom_NotFound(t *testing.T) {
	svc, _, _ := newRoomTestService(nil, nil)
	auth := domain.AuthContext{ProjectID: "proj-a"}
	_, _, err := svc.GetRoom(context.Background(), auth, "missing")
	if err == nil {
		t.Fatal("expected not-found error")
	}
}

func TestRoomService_GetRoom_IncludesState(t *testing.T) {
	room := &domain.Room{ID: "room-1", ProjectID: "proj-a", Status: domain.RoomStatusActive}
	state := &domain.RoomState{RoomID: "room-1", Summary: "hello", State: json.RawMessage(`{"budget":100}`)}
	svc, _, _ := newRoomTestService(room, state)
	auth := domain.AuthContext{ProjectID: "proj-a"}

	gotRoom, gotState, err := svc.GetRoom(context.Background(), auth, "room-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotRoom == nil || gotState == nil {
		t.Fatal("expected room and state")
	}
	if gotState.Summary != "hello" {
		t.Fatalf("state summary = %q, want hello", gotState.Summary)
	}
}

func TestRoomService_ArchiveRoom_NoopIfArchived(t *testing.T) {
	room := &domain.Room{ID: "room-1", ProjectID: "proj-a", Status: domain.RoomStatusArchived}
	svc, rs, _ := newRoomTestService(room, nil)
	auth := domain.AuthContext{ProjectID: "proj-a"}

	got, err := svc.ArchiveRoom(context.Background(), auth, "room-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != domain.RoomStatusArchived {
		t.Fatalf("status = %q", got.Status)
	}
	// No update should have been issued for an already-archived room.
	if rs.room.Status != domain.RoomStatusArchived {
		t.Fatalf("room status mutated unexpectedly")
	}
}

func TestRoomService_ArchiveRoom_SetsStatus(t *testing.T) {
	room := &domain.Room{ID: "room-1", ProjectID: "proj-a", Status: domain.RoomStatusActive}
	svc, rs, _ := newRoomTestService(room, nil)
	auth := domain.AuthContext{ProjectID: "proj-a"}

	got, err := svc.ArchiveRoom(context.Background(), auth, "room-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != domain.RoomStatusArchived {
		t.Fatalf("status = %q, want archived", got.Status)
	}
	if rs.room.Status != domain.RoomStatusArchived {
		t.Fatalf("store status = %q, want archived", rs.room.Status)
	}
}

func TestRoomService_UpdateRoomState_NotFound(t *testing.T) {
	svc, _, _ := newRoomTestService(nil, nil)
	auth := domain.AuthContext{ProjectID: "proj-a"}
	_, err := svc.UpdateRoomState(context.Background(), auth, "missing", map[string]any{"a": 1}, nil)
	if err == nil {
		t.Fatal("expected error for missing room")
	}
}