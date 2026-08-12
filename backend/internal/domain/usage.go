package domain

import "context"

type UsageRecord struct {
	ID           string  `db:"id" json:"id"`
	TenantID     string  `db:"tenant_id" json:"tenant_id"`
	Endpoint     string  `db:"endpoint" json:"endpoint"`
	RoomID       *string `db:"room_id" json:"room_id"`
	TokensUsed   int     `db:"tokens_used" json:"tokens_used"`
	TokensInput  int     `db:"tokens_input" json:"tokens_input"`
	TokensOutput int     `db:"tokens_output" json:"tokens_output"`
	CostInput    float64 `db:"cost_input" json:"cost_input"`
	CostOutput   float64 `db:"cost_output" json:"cost_output"`
	CostTotal    float64 `db:"cost_total" json:"cost_total"`
	CreatedAt    string  `db:"created_at" json:"created_at"`
}

type RoomMetrics struct {
	TotalTokens      int     `json:"total_tokens"`
	TotalCost        float64 `json:"total_cost"`
	AvgTokensPerTurn float64 `json:"avg_tokens_per_turn"`
}

type UsageSummary struct {
	TenantID         string  `json:"tenant_id"`
	TotalRooms       int     `json:"total_rooms"`
	TotalMessages    int     `json:"total_messages"`
	TotalMemories    int     `json:"total_memories"`
	TotalAPICalls    int     `json:"total_api_calls"`
	TotalTokensUsed  int     `json:"total_tokens_used"`
	AvgTokensInput   float64 `json:"avg_tokens_input"`
	AvgTokensOutput  float64 `json:"avg_tokens_output"`
	TotalCostInput   float64 `json:"total_cost_input"`
	TotalCostOutput  float64 `json:"total_cost_output"`
	TotalCostOverall float64 `json:"total_cost_overall"`
}

type UsageRepository interface {
	Record(ctx context.Context, rec *UsageRecord) error
	GetSummary(ctx context.Context, tenantID string) (*UsageSummary, error)
	GetRoomMetrics(ctx context.Context, roomID string) (*RoomMetrics, error)
}

type UsageUsecase interface {
	RecordUsage(ctx context.Context, tenantID, endpoint string, roomID *string, tokensIn, tokensOut int) error
	GetUsageSummary(ctx context.Context, tenantID string) (*UsageSummary, error)
	GetRoomMetrics(ctx context.Context, roomID string) (*RoomMetrics, error)
}
