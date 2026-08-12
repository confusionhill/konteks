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
	query := `INSERT INTO usage_records (tenant_id, endpoint, room_id, tokens_used) VALUES ($1, $2, $3, $4)`
	_, err := r.db.ExecContext(ctx, query, rec.TenantID, rec.Endpoint, rec.RoomID, rec.TokensUsed)
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

	// Count messages (across all rooms for tenant)
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

	// Count API calls and total tokens
	row := r.db.QueryRowContext(ctx,
		`SELECT COALESCE(COUNT(*), 0), COALESCE(SUM(tokens_used), 0) FROM usage_records WHERE tenant_id = $1`, tenantID)
	err = row.Scan(&summary.TotalAPICalls, &summary.TotalTokensUsed)
	if err != nil {
		return nil, err
	}

	return summary, nil
}
