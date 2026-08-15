package domain

import (
	"context"
	"time"
)

const (
	ContextHealthModeStrict        = "strict"
	ContextHealthModeBalanced      = "balanced"
	ContextHealthModeCostOptimized = "cost_optimized"
	ContextHealthModeOff           = "off"

	MemoryStatusActive     = "active"
	MemoryStatusSuperseded = "superseded"
	MemoryStatusFlagged    = "flagged"
)

type ContextHealthPolicy struct {
	Enabled                bool    `json:"enabled"`
	Mode                   string  `json:"mode"`
	DecayHalfLifeDays      int     `json:"decay_half_life_days"`
	MaxEntropyThreshold    float64 `json:"max_entropy_threshold"`
	TurnEvalInterval       int     `json:"turn_eval_interval"`
	DeduplicationThreshold float64 `json:"deduplication_threshold"`
	E2EEEvalLocation       string  `json:"e2ee_eval_location"`
}

type MemoryRevision struct {
	ID            string    `db:"id" json:"id"`
	MemoryID      string    `db:"memory_id" json:"memory_id"`
	RoomID        string    `db:"room_id" json:"room_id"`
	Action        string    `db:"action" json:"action"`
	PreviousState string    `db:"previous_state" json:"previous_state"`
	NewState      string    `db:"new_state" json:"new_state"`
	Reason        string    `db:"reason" json:"reason"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type ContextHealthRepository interface {
	GetProjectPolicy(ctx context.Context, projectID string) (*ContextHealthPolicy, error)
	UpdateProjectPolicy(ctx context.Context, projectID string, policy *ContextHealthPolicy) error
	UpdateMemoryStatus(ctx context.Context, memoryID string, status string, supersededByID *string) error
	LogMemoryRevision(ctx context.Context, rev *MemoryRevision) error
	ListMemoriesByRoom(ctx context.Context, roomID string, includeSuperseded bool) ([]Memory, error)
	RestoreMemory(ctx context.Context, memoryID string) error
	GetMemoryByID(ctx context.Context, memoryID string) (*Memory, error)
}

type ContextHealthUsecase interface {
	GetProjectPolicy(ctx context.Context, projectID string) (*ContextHealthPolicy, error)
	UpdateProjectPolicy(ctx context.Context, projectID string, policy *ContextHealthPolicy) error
	SanitizeIngressMemory(ctx context.Context, projectID string, content string) (entropy float64, isWhitelisted bool, shouldFlag bool, err error)
	ProcessMicroSweep(ctx context.Context, projectID string, roomID string) error
	ListRoomMemories(ctx context.Context, projectID string, roomID string, includeSuperseded bool) ([]Memory, error)
	RestoreMemory(ctx context.Context, projectID string, memoryID string) error
	CleanRoomContextHealth(ctx context.Context, projectID string, roomID string) error
}
