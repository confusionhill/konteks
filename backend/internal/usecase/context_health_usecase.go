package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/dika/contextkit/backend/internal/service"
)

type contextHealthUsecase struct {
	healthRepo       domain.ContextHealthRepository
	roomRepo         domain.RoomRepository
	openRouterClient domain.OpenRouterServiceClient
}

func NewContextHealthUsecase(
	healthRepo domain.ContextHealthRepository,
	roomRepo domain.RoomRepository,
	openRouterClient domain.OpenRouterServiceClient,
) domain.ContextHealthUsecase {
	return &contextHealthUsecase{
		healthRepo:       healthRepo,
		roomRepo:         roomRepo,
		openRouterClient: openRouterClient,
	}
}

func (u *contextHealthUsecase) GetProjectPolicy(ctx context.Context, projectID string) (*domain.ContextHealthPolicy, error) {
	return u.healthRepo.GetProjectPolicy(ctx, projectID)
}

func (u *contextHealthUsecase) UpdateProjectPolicy(ctx context.Context, projectID string, policy *domain.ContextHealthPolicy) error {
	if policy.MaxEntropyThreshold <= 0 {
		policy.MaxEntropyThreshold = 4.8
	}
	if policy.TurnEvalInterval <= 0 {
		policy.TurnEvalInterval = 50
	}
	if policy.DecayHalfLifeDays <= 0 {
		policy.DecayHalfLifeDays = 30
	}
	if policy.Mode == "" {
		policy.Mode = domain.ContextHealthModeBalanced
	}
	return u.healthRepo.UpdateProjectPolicy(ctx, projectID, policy)
}

func (u *contextHealthUsecase) SanitizeIngressMemory(ctx context.Context, projectID string, content string) (float64, bool, bool, error) {
	policy, err := u.healthRepo.GetProjectPolicy(ctx, projectID)
	if err != nil || policy == nil || !policy.Enabled {
		return 0.0, true, false, nil
	}

	entropy := service.CalculateShannonEntropy(content)
	isWhitelisted := service.IsWhitelistedTechnicalContent(content)

	shouldFlag := false
	if !isWhitelisted && entropy > policy.MaxEntropyThreshold {
		shouldFlag = true
		log.Printf("[ContextHealth]: Flagged high-entropy memory payload (entropy: %.2f > threshold: %.2f)", entropy, policy.MaxEntropyThreshold)
	}

	return entropy, isWhitelisted, shouldFlag, nil
}

func (u *contextHealthUsecase) ProcessMicroSweep(ctx context.Context, projectID string, roomID string) error {
	policy, err := u.healthRepo.GetProjectPolicy(ctx, projectID)
	if err != nil || policy == nil || !policy.Enabled {
		return nil
	}

	// Mode capability cliff check: cost_optimized and off skip LLM background contradiction evaluation
	if policy.Mode == domain.ContextHealthModeOff || policy.Mode == domain.ContextHealthModeCostOptimized {
		log.Printf("[ContextHealth]: Project %s mode '%s' skips LLM contradiction evaluation", projectID, policy.Mode)
		return nil
	}

	memories, err := u.healthRepo.ListMemoriesByRoom(ctx, roomID, false)
	if err != nil || len(memories) < 2 {
		return err
	}

	// Evaluate candidate pairs (most recent memory vs historical active memories)
	latestMem := memories[0]
	for i := 1; i < len(memories); i++ {
		olderMem := memories[i]
		if olderMem.Status != domain.MemoryStatusActive {
			continue
		}

		isContradiction, confidence, errEval := u.openRouterClient.EvaluateContradiction(ctx, olderMem.Content, latestMem.Content)
		if errEval != nil {
			log.Printf("[ContextHealth Sweep Error]: contradiction evaluation failed: %v", errEval)
			continue
		}

		if isContradiction && confidence >= 0.90 {
			log.Printf("[ContextHealth Sweep]: Fact '%s' superseded by new fact '%s' (confidence: %.2f)", olderMem.ID, latestMem.ID, confidence)
			
			errUpdate := u.healthRepo.UpdateMemoryStatus(ctx, olderMem.ID, domain.MemoryStatusSuperseded, &latestMem.ID)
			if errUpdate == nil {
				prevJSON, _ := json.Marshal(olderMem)
				newJSON, _ := json.Marshal(map[string]interface{}{
					"status":           domain.MemoryStatusSuperseded,
					"superseded_by_id": latestMem.ID,
				})

				_ = u.healthRepo.LogMemoryRevision(ctx, &domain.MemoryRevision{
					MemoryID:      olderMem.ID,
					RoomID:        roomID,
					Action:        "superseded",
					PreviousState: string(prevJSON),
					NewState:      string(newJSON),
					Reason:        fmt.Sprintf("Superseded by memory %s (confidence: %.2f)", latestMem.ID, confidence),
				})
			}
		}
	}

	return nil
}

func (u *contextHealthUsecase) ListRoomMemories(ctx context.Context, projectID string, roomID string, includeSuperseded bool) ([]domain.Memory, error) {
	_, err := u.roomRepo.GetByID(ctx, projectID, roomID)
	if err != nil {
		return nil, err
	}
	return u.healthRepo.ListMemoriesByRoom(ctx, roomID, includeSuperseded)
}

func (u *contextHealthUsecase) RestoreMemory(ctx context.Context, projectID string, memoryID string) error {
	mem, err := u.healthRepo.GetMemoryByID(ctx, memoryID)
	if err != nil || mem == nil {
		return fmt.Errorf("memory not found: %w", err)
	}

	if mem.ProjectID != projectID {
		return fmt.Errorf("unauthorized memory access")
	}

	prevJSON, _ := json.Marshal(mem)
	errRestore := u.healthRepo.RestoreMemory(ctx, memoryID)
	if errRestore != nil {
		return errRestore
	}

	roomID := ""
	if mem.RoomID != nil {
		roomID = *mem.RoomID
	}

	newJSON, _ := json.Marshal(map[string]interface{}{"status": domain.MemoryStatusActive})
	_ = u.healthRepo.LogMemoryRevision(ctx, &domain.MemoryRevision{
		MemoryID:      memoryID,
		RoomID:        roomID,
		Action:        "restored",
		PreviousState: string(prevJSON),
		NewState:      string(newJSON),
		Reason:        "Manually restored via Context Health API",
	})

	return nil
}

func (u *contextHealthUsecase) CleanRoomContextHealth(ctx context.Context, projectID string, roomID string) error {
	return u.ProcessMicroSweep(ctx, projectID, roomID)
}
