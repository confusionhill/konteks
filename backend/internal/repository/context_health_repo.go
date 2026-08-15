package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

type contextHealthRepository struct {
	db *sqlx.DB
}

func NewContextHealthRepository(db *sqlx.DB) domain.ContextHealthRepository {
	return &contextHealthRepository{db: db}
}

func (r *contextHealthRepository) GetProjectPolicy(ctx context.Context, projectID string) (*domain.ContextHealthPolicy, error) {
	query := `
		SELECT context_health_policy
		FROM projects
		WHERE id = $1 AND deleted_at IS NULL
	`
	var policyJSON []byte
	err := r.db.GetContext(ctx, &policyJSON, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch project context health policy: %w", err)
	}

	policy := &domain.ContextHealthPolicy{
		Enabled:                true,
		Mode:                   domain.ContextHealthModeBalanced,
		DecayHalfLifeDays:      30,
		MaxEntropyThreshold:    4.8,
		TurnEvalInterval:       50,
		DeduplicationThreshold: 0.95,
		E2EEEvalLocation:       "server_distance_only",
	}

	if len(policyJSON) > 0 {
		_ = json.Unmarshal(policyJSON, policy)
	}

	return policy, nil
}

func (r *contextHealthRepository) UpdateProjectPolicy(ctx context.Context, projectID string, policy *domain.ContextHealthPolicy) error {
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal policy: %w", err)
	}

	query := `
		UPDATE projects
		SET context_health_policy = $1, updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL
	`
	res, err := r.db.ExecContext(ctx, query, policyJSON, projectID)
	if err != nil {
		return fmt.Errorf("failed to update project context health policy: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("project not found or deleted")
	}

	return nil
}

func (r *contextHealthRepository) UpdateMemoryStatus(ctx context.Context, memoryID string, status string, supersededByID *string) error {
	query := `
		UPDATE memories
		SET status = $1, superseded_by_id = $2
		WHERE id = $3
	`
	_, err := r.db.ExecContext(ctx, query, status, supersededByID, memoryID)
	if err != nil {
		return fmt.Errorf("failed to update memory status: %w", err)
	}
	return nil
}

func (r *contextHealthRepository) LogMemoryRevision(ctx context.Context, rev *domain.MemoryRevision) error {
	query := `
		INSERT INTO memory_revisions (memory_id, room_id, action, previous_state, new_state, reason)
		VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6)
		RETURNING id, created_at
	`
	type Ins struct {
		ID        string `db:"id"`
		CreatedAt string `db:"created_at"`
	}
	var ins Ins
	err := r.db.GetContext(ctx, &ins, query,
		rev.MemoryID,
		rev.RoomID,
		rev.Action,
		rev.PreviousState,
		rev.NewState,
		rev.Reason,
	)
	if err != nil {
		return fmt.Errorf("failed to log memory revision: %w", err)
	}

	rev.ID = ins.ID
	return nil
}

func (r *contextHealthRepository) ListMemoriesByRoom(ctx context.Context, roomID string, includeSuperseded bool) ([]domain.Memory, error) {
	var query string
	if includeSuperseded {
		query = `
			SELECT id, project_id, room_id, external_user_id, content, embedding::text, metadata, created_at, status, superseded_by_id, shannon_entropy, is_whitelisted
			FROM memories
			WHERE room_id = $1
			ORDER BY created_at DESC
		`
	} else {
		query = `
			SELECT id, project_id, room_id, external_user_id, content, embedding::text, metadata, created_at, status, superseded_by_id, shannon_entropy, is_whitelisted
			FROM memories
			WHERE room_id = $1 AND status = 'active'
			ORDER BY created_at DESC
		`
	}

	type dbMemory struct {
		ID             string  `db:"id"`
		ProjectID      string  `db:"project_id"`
		RoomID         *string `db:"room_id"`
		ExternalUserID *string `db:"external_user_id"`
		Content        string  `db:"content"`
		EmbeddingText  string  `db:"embedding"`
		Metadata       string  `db:"metadata"`
		CreatedAt      string  `db:"created_at"`
		Status         string  `db:"status"`
		SupersededByID *string `db:"superseded_by_id"`
		ShannonEntropy float64 `db:"shannon_entropy"`
		IsWhitelisted  bool    `db:"is_whitelisted"`
	}

	var rows []dbMemory
	err := r.db.SelectContext(ctx, &rows, query, roomID)
	if err != nil {
		return nil, fmt.Errorf("failed to list memories by room: %w", err)
	}

	memories := make([]domain.Memory, len(rows))
	for i, row := range rows {
		emb, _ := stringToFloat32Array(row.EmbeddingText)
		memories[i] = domain.Memory{
			ID:             row.ID,
			ProjectID:      row.ProjectID,
			RoomID:         row.RoomID,
			ExternalUserID: row.ExternalUserID,
			Content:        row.Content,
			Embedding:      emb,
			Metadata:       row.Metadata,
			Status:         row.Status,
			SupersededByID: row.SupersededByID,
			ShannonEntropy: row.ShannonEntropy,
			IsWhitelisted:  row.IsWhitelisted,
		}
	}

	return memories, nil
}

func (r *contextHealthRepository) RestoreMemory(ctx context.Context, memoryID string) error {
	query := `
		UPDATE memories
		SET status = 'active', superseded_by_id = NULL
		WHERE id = $1
	`
	res, err := r.db.ExecContext(ctx, query, memoryID)
	if err != nil {
		return fmt.Errorf("failed to restore memory: %w", err)
	}

	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("memory not found")
	}

	return nil
}

func (r *contextHealthRepository) GetMemoryByID(ctx context.Context, memoryID string) (*domain.Memory, error) {
	query := `
		SELECT id, project_id, room_id, external_user_id, content, embedding::text, metadata, created_at, status, superseded_by_id, shannon_entropy, is_whitelisted
		FROM memories
		WHERE id = $1
	`
	type dbMemory struct {
		ID             string  `db:"id"`
		ProjectID      string  `db:"project_id"`
		RoomID         *string `db:"room_id"`
		ExternalUserID *string `db:"external_user_id"`
		Content        string  `db:"content"`
		EmbeddingText  string  `db:"embedding"`
		Metadata       string  `db:"metadata"`
		CreatedAt      string  `db:"created_at"`
		Status         string  `db:"status"`
		SupersededByID *string `db:"superseded_by_id"`
		ShannonEntropy float64 `db:"shannon_entropy"`
		IsWhitelisted  bool    `db:"is_whitelisted"`
	}

	var row dbMemory
	err := r.db.GetContext(ctx, &row, query, memoryID)
	if err != nil {
		return nil, fmt.Errorf("memory not found: %w", err)
	}

	emb, _ := stringToFloat32Array(row.EmbeddingText)
	return &domain.Memory{
		ID:             row.ID,
		ProjectID:      row.ProjectID,
		RoomID:         row.RoomID,
		ExternalUserID: row.ExternalUserID,
		Content:        row.Content,
		Embedding:      emb,
		Metadata:       row.Metadata,
		Status:         row.Status,
		SupersededByID: row.SupersededByID,
		ShannonEntropy: row.ShannonEntropy,
		IsWhitelisted:  row.IsWhitelisted,
	}, nil
}
