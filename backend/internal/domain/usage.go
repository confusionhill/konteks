package domain

import "context"

type UsageRecord struct {
	ID         string `db:"id" json:"id"`
	TenantID   string `db:"tenant_id" json:"tenant_id"`
	Endpoint   string `db:"endpoint" json:"endpoint"`
	RoomID     *string `db:"room_id" json:"room_id"`
	TokensUsed int    `db:"tokens_used" json:"tokens_used"`
	CreatedAt  string `db:"created_at" json:"created_at"`
}

type UsageSummary struct {
	TenantID        string `json:"tenant_id"`
	TotalRooms      int    `json:"total_rooms"`
	TotalMessages   int    `json:"total_messages"`
	TotalMemories   int    `json:"total_memories"`
	TotalAPICalls   int    `json:"total_api_calls"`
	TotalTokensUsed int    `json:"total_tokens_used"`
}

type UsageRepository interface {
	Record(ctx context.Context, rec *UsageRecord) error
	GetSummary(ctx context.Context, tenantID string) (*UsageSummary, error)
}

type UsageUsecase interface {
	RecordUsage(ctx context.Context, tenantID, endpoint string, roomID *string, tokens int) error
	GetUsageSummary(ctx context.Context, tenantID string) (*UsageSummary, error)
}
