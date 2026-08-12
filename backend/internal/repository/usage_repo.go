package repository

import (
	"context"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

type usageRepository struct {
	db *sqlx.DB
}

func NewUsageRepository(db *sqlx.DB) domain.UsageRepository {
	return &usageRepository{db: db}
}

func (r *usageRepository) Record(ctx context.Context, rec *domain.UsageRecord) error {
	query := `
		INSERT INTO usage_records (
			tenant_id, endpoint, room_id, tokens_used, 
			tokens_input, tokens_output, cost_input, cost_output, cost_total
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`
	_, err := r.db.ExecContext(ctx, query,
		rec.TenantID, rec.Endpoint, rec.RoomID, rec.TokensUsed,
		rec.TokensInput, rec.TokensOutput, rec.CostInput, rec.CostOutput, rec.CostTotal,
	)
	return err
}

func (r *usageRepository) GetSummary(ctx context.Context, tenantID string) (*domain.UsageSummary, error) {
	summary := &domain.UsageSummary{TenantID: tenantID}

	// Count rooms
	err := r.db.GetContext(ctx, &summary.TotalRooms,
		`SELECT COUNT(*) FROM rooms WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, err
	}

	// Count messages
	err = r.db.GetContext(ctx, &summary.TotalMessages,
		`SELECT COUNT(*) FROM messages m JOIN rooms r ON m.room_id = r.id WHERE r.tenant_id = $1`, tenantID)
	if err != nil {
		return nil, err
	}

	// Count memories
	err = r.db.GetContext(ctx, &summary.TotalMemories,
		`SELECT COUNT(*) FROM memories WHERE tenant_id = $1`, tenantID)
	if err != nil {
		return nil, err
	}

	// Calculate usage metrics and cost aggregations
	queryUsage := `
		SELECT 
			COALESCE(COUNT(*), 0) AS total_api_calls,
			COALESCE(SUM(tokens_used), 0) AS total_tokens_used,
			COALESCE(AVG(tokens_input), 0) AS avg_tokens_input,
			COALESCE(AVG(tokens_output), 0) AS avg_tokens_output,
			COALESCE(SUM(cost_input), 0) AS total_cost_input,
			COALESCE(SUM(cost_output), 0) AS total_cost_output,
			COALESCE(SUM(cost_total), 0) AS total_cost_overall
		FROM usage_records 
		WHERE tenant_id = $1
	`
	row := r.db.QueryRowContext(ctx, queryUsage, tenantID)
	err = row.Scan(
		&summary.TotalAPICalls,
		&summary.TotalTokensUsed,
		&summary.AvgTokensInput,
		&summary.AvgTokensOutput,
		&summary.TotalCostInput,
		&summary.TotalCostOutput,
		&summary.TotalCostOverall,
	)
	if err != nil {
		return nil, err
	}

	return summary, nil
}
