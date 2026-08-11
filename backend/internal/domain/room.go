package domain

import (
	"context"
	"encoding/json"
)

type Room struct {
	ID             string `db:"id" json:"id"`
	TenantID       string `db:"tenant_id" json:"tenant_id"`
	ExternalUserID string `db:"external_user_id" json:"external_user_id"`
	CreatedAt      string `db:"created_at" json:"created_at"`
	UpdatedAt      string `db:"updated_at" json:"updated_at"`
}

type RoomState struct {
	RoomID      string          `db:"room_id" json:"room_id"`
	Summary     string          `db:"summary" json:"summary"`
	ActiveTopic string          `db:"active_topic" json:"active_topic"`
	State       json.RawMessage `db:"state" json:"state"`
	UpdatedAt   string          `db:"updated_at" json:"updated_at"`
}

type RoomRepository interface {
	Create(ctx context.Context, tenantID string, externalUserID string) (*Room, error)
	GetByID(ctx context.Context, tenantID string, roomID string) (*Room, error)
	Delete(ctx context.Context, tenantID string, roomID string) error
	GetState(ctx context.Context, roomID string) (*RoomState, error)
	UpsertState(ctx context.Context, state *RoomState) error
}

type RoomUsecase interface {
	CreateRoom(ctx context.Context, tenantID string, externalUserID string) (*Room, error)
	GetRoomState(ctx context.Context, tenantID string, roomID string) (*RoomState, error)
	DeleteRoom(ctx context.Context, tenantID string, roomID string) error
}
