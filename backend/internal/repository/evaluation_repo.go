package repository

import (
	"context"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

type evaluationRepo struct {
	db *sqlx.DB
}

func NewEvaluationRepository(db *sqlx.DB) domain.EvaluationRepository {
	return &evaluationRepo{db: db}
}

func (r *evaluationRepo) Create(ctx context.Context, eval *domain.ContextEvaluation) error {
	q := `INSERT INTO context_evaluations 
		(id, project_id, room_id, query, context_snapshot, relevance_score, completeness_score, token_efficiency_score, reasoning, is_new_topic)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`
	_, err := r.db.ExecContext(ctx, q, eval.ID, eval.ProjectID, eval.RoomID, eval.Query, eval.ContextSnapshot, eval.RelevanceScore, eval.CompletenessScore, eval.TokenEfficiencyScore, eval.Reasoning, eval.IsNewTopic)
	return err
}

func (r *evaluationRepo) GetByRoomID(ctx context.Context, roomID string) ([]domain.ContextEvaluation, error) {
	q := `SELECT id, project_id, room_id, query, context_snapshot, relevance_score, completeness_score, token_efficiency_score, reasoning, is_new_topic, created_at
		FROM context_evaluations WHERE room_id = $1 ORDER BY created_at DESC`
	rows, err := r.db.QueryxContext(ctx, q, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var evals []domain.ContextEvaluation
	for rows.Next() {
		var e domain.ContextEvaluation
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.RoomID, &e.Query, &e.ContextSnapshot, &e.RelevanceScore, &e.CompletenessScore, &e.TokenEfficiencyScore, &e.Reasoning, &e.IsNewTopic, &e.CreatedAt); err != nil {
			return nil, err
		}
		evals = append(evals, e)
	}
	return evals, nil
}

func (r *evaluationRepo) GetByID(ctx context.Context, evalID string) (*domain.ContextEvaluation, error) {
	q := `SELECT id, project_id, room_id, query, context_snapshot, relevance_score, completeness_score, token_efficiency_score, reasoning, is_new_topic, created_at
		FROM context_evaluations WHERE id = $1`
	var e domain.ContextEvaluation
	err := r.db.QueryRowxContext(ctx, q, evalID).Scan(&e.ID, &e.ProjectID, &e.RoomID, &e.Query, &e.ContextSnapshot, &e.RelevanceScore, &e.CompletenessScore, &e.TokenEfficiencyScore, &e.Reasoning, &e.IsNewTopic, &e.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *evaluationRepo) GetByProjectID(ctx context.Context, projectID string, limit int, offset int) ([]domain.ContextEvaluation, int, error) {
	countQ := `SELECT COUNT(*) FROM context_evaluations WHERE project_id = $1`
	var total int
	if err := r.db.GetContext(ctx, &total, countQ, projectID); err != nil {
		return nil, 0, err
	}

	q := `SELECT id, project_id, room_id, query, context_snapshot, relevance_score, completeness_score, token_efficiency_score, reasoning, is_new_topic, created_at
		FROM context_evaluations 
		WHERE project_id = $1 
		ORDER BY created_at DESC 
		LIMIT $2 OFFSET $3`
	rows, err := r.db.QueryxContext(ctx, q, projectID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var evals []domain.ContextEvaluation
	for rows.Next() {
		var e domain.ContextEvaluation
		if err := rows.Scan(&e.ID, &e.ProjectID, &e.RoomID, &e.Query, &e.ContextSnapshot, &e.RelevanceScore, &e.CompletenessScore, &e.TokenEfficiencyScore, &e.Reasoning, &e.IsNewTopic, &e.CreatedAt); err != nil {
			return nil, 0, err
		}
		evals = append(evals, e)
	}
	return evals, total, nil
}

func (r *evaluationRepo) GetSummary(ctx context.Context, projectID string) (*domain.EvaluationSummary, error) {
	q := `SELECT 
			COALESCE(AVG(relevance_score), 0),
			COALESCE(AVG(completeness_score), 0),
			COALESCE(AVG(token_efficiency_score), 0),
			COUNT(id)
		FROM context_evaluations
		WHERE project_id = $1`
	var sum domain.EvaluationSummary
	err := r.db.QueryRowxContext(ctx, q, projectID).Scan(&sum.AvgRelevance, &sum.AvgCompleteness, &sum.AvgTokenEfficiency, &sum.TotalEvaluations)
	if err != nil {
		return nil, err
	}
	return &sum, nil
}

