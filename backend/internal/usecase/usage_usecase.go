package usecase

import (
	"context"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/dika/contextkit/backend/internal/service"
)

type usageUsecase struct {
	usageRepo domain.UsageRepository
}

func NewUsageUsecase(usageRepo domain.UsageRepository) domain.UsageUsecase {
	return &usageUsecase{usageRepo: usageRepo}
}

func (u *usageUsecase) RecordUsage(ctx context.Context, projectID, endpoint string, roomID *string, tokensIn, tokensOut int) error {
	costIn, costOut, costTotal := service.CalculateCost(tokensIn, tokensOut)
	rec := &domain.UsageRecord{
		ProjectID:    projectID,
		Endpoint:     endpoint,
		RoomID:       roomID,
		TokensUsed:   tokensIn + tokensOut,
		TokensInput:  tokensIn,
		TokensOutput: tokensOut,
		CostInput:    costIn,
		CostOutput:   costOut,
		CostTotal:    costTotal,
	}
	return u.usageRepo.Record(ctx, rec)
}

func (u *usageUsecase) GetUsageSummary(ctx context.Context, projectID string) (*domain.UsageSummary, error) {
	return u.usageRepo.GetSummary(ctx, projectID)
}

func (u *usageUsecase) GetRoomMetrics(ctx context.Context, roomID string) (*domain.RoomMetrics, error) {
	return u.usageRepo.GetRoomMetrics(ctx, roomID)
}
