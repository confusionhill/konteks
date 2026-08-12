package usecase

import (
	"context"

	"github.com/dika/contextkit/backend/internal/domain"
)

type usageUsecase struct {
	usageRepo domain.UsageRepository
}

func NewUsageUsecase(usageRepo domain.UsageRepository) domain.UsageUsecase {
	return &usageUsecase{usageRepo: usageRepo}
}

func (u *usageUsecase) RecordUsage(ctx context.Context, tenantID, endpoint string, roomID *string, tokens int) error {
	rec := &domain.UsageRecord{
		TenantID:   tenantID,
		Endpoint:   endpoint,
		RoomID:     roomID,
		TokensUsed: tokens,
	}
	return u.usageRepo.Record(ctx, rec)
}

func (u *usageUsecase) GetUsageSummary(ctx context.Context, tenantID string) (*domain.UsageSummary, error) {
	return u.usageRepo.GetSummary(ctx, tenantID)
}
