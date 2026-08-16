package repository

import (
	"context"
	"fmt"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

type pgvectorStore struct {
	db *sqlx.DB
}

// NewPgvectorStore creates a VectorStore implementation using PostgreSQL pgvector.
func NewPgvectorStore(db *sqlx.DB) domain.VectorStore {
	return &pgvectorStore{db: db}
}

func (s *pgvectorStore) UpsertMemoryVector(ctx context.Context, memory *domain.Memory) error {
	// Vector is already stored in memories table via memoryRepo in pgvector mode
	return nil
}

func (s *pgvectorStore) SearchMemoryVectors(ctx context.Context, projectID string, roomID string, externalUserID *string, queryEmbedding []float32, limit int) ([]domain.VectorMatch, error) {
	embeddingStr := float32ArrayToString(queryEmbedding)

	var query string
	var args []any

	if externalUserID != nil && *externalUserID != "" {
		query = `
			SELECT id, 1 - (embedding <=> $3::vector) AS score
			FROM memories
			WHERE project_id = $1 AND (room_id = $2 OR external_user_id = $4) AND status = 'active' AND embedding IS NOT NULL
			ORDER BY embedding <=> $3::vector ASC
			LIMIT $5
		`
		args = []any{projectID, roomID, embeddingStr, *externalUserID, limit}
	} else {
		query = `
			SELECT id, 1 - (embedding <=> $2::vector) AS score
			FROM memories
			WHERE project_id = $1 AND room_id = $3 AND status = 'active' AND embedding IS NOT NULL
			ORDER BY embedding <=> $2::vector ASC
			LIMIT $4
		`
		args = []any{projectID, embeddingStr, roomID, limit}
	}

	type dbMatch struct {
		ID    string  `db:"id"`
		Score float64 `db:"score"`
	}

	var rows []dbMatch
	err := s.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, err
	}

	matches := make([]domain.VectorMatch, len(rows))
	for i, r := range rows {
		matches[i] = domain.VectorMatch{
			ID:    r.ID,
			Score: r.Score,
		}
	}
	return matches, nil
}

func (s *pgvectorStore) DeleteMemoryVectors(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query, args, err := sqlx.In("UPDATE memories SET embedding = NULL WHERE id IN (?)", ids)
	if err != nil {
		return err
	}
	query = s.db.Rebind(query)
	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *pgvectorStore) GetVectorByID(ctx context.Context, id string) ([]float32, error) {
	query := `SELECT embedding::text FROM memories WHERE id = $1`
	var embStr string
	err := s.db.GetContext(ctx, &embStr, query, id)
	if err != nil {
		return nil, err
	}
	return stringToFloat32Array(embStr)
}

func (s *pgvectorStore) UpsertArtifactChunkVectors(ctx context.Context, chunks []domain.ArtifactChunk, groupName string) error {
	// Chunks vectors are stored in artifact_chunks table in pgvector mode
	return nil
}

func (s *pgvectorStore) SearchArtifactChunkVectors(ctx context.Context, projectID string, queryEmbedding []float32, limit int, excludedArtifactIDs []string, excludedGroupNames []string) ([]domain.VectorMatch, error) {
	embeddingStr := float32ArrayToString(queryEmbedding)

	query := `
		SELECT ac.id, 1 - (ac.embedding <=> $2::vector) AS score
		FROM artifact_chunks ac
		JOIN artifacts a ON ac.artifact_id = a.id
		LEFT JOIN artifact_groups ag ON a.group_id = ag.id
		WHERE a.project_id = $1 AND ac.embedding IS NOT NULL
	`
	args := []any{projectID, embeddingStr}
	argSeq := 3

	if len(excludedArtifactIDs) > 0 {
		query += fmt.Sprintf(" AND a.id NOT IN ($%d)", argSeq)
		args = append(args, excludedArtifactIDs)
		argSeq++
	}

	if len(excludedGroupNames) > 0 {
		query += fmt.Sprintf(" AND (ag.name IS NULL OR ag.name NOT IN ($%d))", argSeq)
		args = append(args, excludedGroupNames)
		argSeq++
	}

	query += fmt.Sprintf(" ORDER BY ac.embedding <=> $2::vector ASC LIMIT $%d", argSeq)
	args = append(args, limit)

	query, args, err := sqlx.In(query, args...)
	if err != nil {
		return nil, err
	}
	query = s.db.Rebind(query)

	type dbMatch struct {
		ID    string  `db:"id"`
		Score float64 `db:"score"`
	}

	var rows []dbMatch
	err = s.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, err
	}

	matches := make([]domain.VectorMatch, len(rows))
	for i, r := range rows {
		matches[i] = domain.VectorMatch{
			ID:    r.ID,
			Score: r.Score,
		}
	}
	return matches, nil
}

func (s *pgvectorStore) DeleteArtifactChunkVectors(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	query, args, err := sqlx.In("UPDATE artifact_chunks SET embedding = NULL WHERE id IN (?)", ids)
	if err != nil {
		return err
	}
	query = s.db.Rebind(query)
	_, err = s.db.ExecContext(ctx, query, args...)
	return err
}
