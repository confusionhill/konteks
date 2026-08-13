package domain

import (
	"context"
	"encoding/json"
	"time"
)

const (
	RoomStatusActive    = "active"
	RoomStatusArchived  = "archived"
	RoomStatusArchiving = "archiving"
	RoomStatusHydrating = "hydrating"
)

type Room struct {
	ID             string          `db:"id" json:"id"`
	ProjectID      string          `db:"project_id" json:"project_id"`
	ExternalUserID string          `db:"external_user_id" json:"external_user_id"`
	CreatedAt      string          `db:"created_at" json:"created_at"`
	UpdatedAt      string          `db:"updated_at" json:"updated_at"`
	StateSchema    json.RawMessage `db:"state_schema" json:"state_schema,omitempty"`
	DefaultPolicy  *ContextPolicy  `db:"default_policy" json:"default_policy,omitempty"`
	Status         string          `db:"status" json:"status"`
	ArchiveS3Path  *string         `db:"archive_s3_path" json:"archive_s3_path,omitempty"`
}

type RoomState struct {
	RoomID            string            `db:"room_id" json:"room_id"`
	Summary           string            `db:"summary" json:"summary"`
	ActiveTopic       string            `db:"active_topic" json:"active_topic"`
	State             json.RawMessage   `db:"state" json:"state"`
	References        map[string]string `json:"references,omitempty"`
	Entities          []Entity          `json:"entities,omitempty"`
	LastSummarizedSeq int64             `db:"last_summarized_seq" json:"last_summarized_seq"`
	UpdatedAt         string            `db:"updated_at" json:"updated_at"`
}

type RoomRepository interface {
	Create(ctx context.Context, projectID string, externalUserID string, stateSchema json.RawMessage, defaultPolicy *ContextPolicy) (*Room, error)
	GetByID(ctx context.Context, projectID string, roomID string) (*Room, error)
	Update(ctx context.Context, projectID string, roomID string, stateSchema json.RawMessage, defaultPolicy *ContextPolicy) (*Room, error)
	Delete(ctx context.Context, projectID string, roomID string) error
	GetState(ctx context.Context, roomID string) (*RoomState, error)
	UpsertState(ctx context.Context, state *RoomState) error
	ListWithOverview(ctx context.Context, projectID string, search string, status string, sortBy string, sortOrder string, minMessages int, limit int, offset int) ([]RoomOverview, int, error)
	ListRoomsForArchival(ctx context.Context, inactiveDuration time.Duration, roomID string) ([]Room, error)
	UpdateStatus(ctx context.Context, roomID string, status string) error
	ArchiveRoomTx(ctx context.Context, roomID string, archiveS3Path string) error
	HydrateRoomTx(ctx context.Context, roomID string, state *RoomState, messages []Message, memories []Memory) error
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
	Status           string  `db:"status" json:"status"`
	ArchiveS3Path    *string `db:"archive_s3_path" json:"archive_s3_path,omitempty"`
}

type RoomUsecase interface {
	CreateRoom(ctx context.Context, projectID string, externalUserID string, stateSchema json.RawMessage, defaultPolicy *ContextPolicy) (*Room, error)
	GetRoomState(ctx context.Context, projectID string, roomID string) (*RoomState, error)
	UpdateRoom(ctx context.Context, projectID string, roomID string, stateSchema json.RawMessage, defaultPolicy *ContextPolicy) (*Room, error)
	DeleteRoom(ctx context.Context, projectID string, roomID string) error
	ListRoomsOverview(ctx context.Context, projectID string, search string, status string, sortBy string, sortOrder string, minMessages int, limit int, offset int) ([]RoomOverview, int, error)
}
