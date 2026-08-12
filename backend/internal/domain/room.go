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
	RoomID      string            `db:"room_id" json:"room_id"`
	Summary     string            `db:"summary" json:"summary"`
	ActiveTopic string            `db:"active_topic" json:"active_topic"`
	State       json.RawMessage   `db:"state" json:"state"`
	References  map[string]string `json:"references,omitempty"`
	Entities    []Entity          `json:"entities,omitempty"`
	UpdatedAt   string            `db:"updated_at" json:"updated_at"`
}

type RoomRepository interface {
	Create(ctx context.Context, tenantID string, externalUserID string) (*Room, error)
	GetByID(ctx context.Context, tenantID string, roomID string) (*Room, error)
	Delete(ctx context.Context, tenantID string, roomID string) error
	GetState(ctx context.Context, roomID string) (*RoomState, error)
	UpsertState(ctx context.Context, state *RoomState) error
	ListWithOverview(ctx context.Context, tenantID string) ([]RoomOverview, error)
}

type RoomOverview struct {
	ID               string  `db:"id" json:"id"`
	ExternalUserID   string  `db:"external_user_id" json:"external_user_id"`
	ActiveTopic      string  `db:"active_topic" json:"active_topic"`
	MessageCount     int     `db:"message_count" json:"message_count"`
	TotalTokensUsed  int     `db:"total_tokens_used" json:"total_tokens_used"`
	TotalCost        float64 `db:"total_cost" json:"total_cost"`
	AvgTokensPerTurn float64 `db:"avg_tokens_per_turn" json:"avg_tokens_per_turn"`
	LastActiveAt     string  `db:"last_active_at" json:"last_active_at"`
}

type RoomUsecase interface {
	CreateRoom(ctx context.Context, tenantID string, externalUserID string) (*Room, error)
	GetRoomState(ctx context.Context, tenantID string, roomID string) (*RoomState, error)
	DeleteRoom(ctx context.Context, tenantID string, roomID string) error
	ListRoomsOverview(ctx context.Context, tenantID string) ([]RoomOverview, error)
}
