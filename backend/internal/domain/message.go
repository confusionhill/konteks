package domain

import "context"

type Message struct {
	ID                 string   `db:"id" json:"id"`
	RoomID             string   `db:"room_id" json:"room_id"`
	Role               string   `db:"role" json:"role"`
	Content            string   `db:"content" json:"content"`
	Sequence           int64    `db:"sequence" json:"sequence"`
	CreatedAt          string   `db:"created_at" json:"created_at"`
	TokensInput        *int     `db:"tokens_input" json:"tokens_input,omitempty"`
	TokensOutput       *int     `db:"tokens_output" json:"tokens_output,omitempty"`
	CostTotal          *float64 `db:"cost_total" json:"cost_total,omitempty"`
	IsEncrypted        bool     `db:"is_encrypted" json:"is_encrypted"`
	EncryptionMetadata *string  `db:"encryption_metadata" json:"encryption_metadata,omitempty"`
}

type Memory struct {
	ID                 string    `db:"id" json:"id"`
	ProjectID          string    `db:"project_id" json:"project_id"`
	RoomID             *string   `db:"room_id" json:"room_id"`
	ExternalUserID     *string   `db:"external_user_id" json:"external_user_id"`
	Content            string    `db:"content" json:"content"`
	Embedding          []float32 `db:"embedding" json:"embedding,omitempty"`
	Metadata           string    `db:"metadata" json:"metadata"`
	IsEncrypted        bool      `db:"is_encrypted" json:"is_encrypted"`
	EncryptionMetadata *string   `db:"encryption_metadata" json:"encryption_metadata,omitempty"`
	Status             string    `db:"status" json:"status"`
	SupersededByID     *string   `db:"superseded_by_id" json:"superseded_by_id,omitempty"`
	ShannonEntropy     float64   `db:"shannon_entropy" json:"shannon_entropy"`
	IsWhitelisted      bool      `db:"is_whitelisted" json:"is_whitelisted"`
}

type ProjectLogEntry struct {
	ID                 string   `db:"id" json:"id"`
	RoomID             string   `db:"room_id" json:"room_id"`
	ExternalUserID     string   `db:"external_user_id" json:"external_user_id"`
	Role               string   `db:"role" json:"role"`
	Content            string   `db:"content" json:"content"`
	Sequence           int64    `db:"sequence" json:"sequence"`
	CreatedAt          string   `db:"created_at" json:"created_at"`
	TokensInput        *int     `db:"tokens_input" json:"tokens_input,omitempty"`
	TokensOutput       *int     `db:"tokens_output" json:"tokens_output,omitempty"`
	CostTotal          *float64 `db:"cost_total" json:"cost_total,omitempty"`
	IsEncrypted        bool     `db:"is_encrypted" json:"is_encrypted"`
	EncryptionMetadata *string  `db:"encryption_metadata" json:"encryption_metadata,omitempty"`
}

type MessageRepository interface {
	Create(ctx context.Context, msg *Message) error
	ListByRoom(ctx context.Context, roomID string, limit int, beforeSeq int64) ([]Message, error)
	GetMaxSequence(ctx context.Context, roomID string) (int64, error)
	ListProjectLogs(ctx context.Context, projectID string, roomID string, externalUserID string, search string, role string, sortBy string, sortOrder string, limit int, offset int) ([]ProjectLogEntry, int, error)
}

type MemoryRepository interface {
	Create(ctx context.Context, memory *Memory) error
	SearchSemantic(ctx context.Context, roomID string, externalUserID *string, queryEmbedding []float32, freshnessWeight float64, limit int) ([]Memory, error)
	ListByRoom(ctx context.Context, roomID string) ([]Memory, error)
}

type MessageUsecase interface {
	CreateMessage(ctx context.Context, projectID string, roomID string, role string, content string, isEncrypted bool, encryptionMetadata *string) (*Message, error)
	ListMessages(ctx context.Context, projectID string, roomID string, limit int, beforeSeq int64) ([]Message, error)
	ListProjectLogs(ctx context.Context, projectID string, roomID string, externalUserID string, search string, role string, sortBy string, sortOrder string, limit int, offset int) ([]ProjectLogEntry, int, error)
}
