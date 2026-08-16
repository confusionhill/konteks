package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

type memoryRepository struct {
	db *sqlx.DB
}

func NewMemoryRepository(db *sqlx.DB) domain.MemoryRepository {
	return &memoryRepository{db: db}
}

// Helper to convert float32 array to PostgreSQL pgvector string format: [0.1,0.2,...]
func float32ArrayToString(arr []float32) string {
	var builder strings.Builder
	builder.WriteByte('[')
	for i, f := range arr {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(strconv.FormatFloat(float64(f), 'f', -1, 32))
	}
	builder.WriteByte(']')
	return builder.String()
}

// Helper to parse PostgreSQL pgvector string back to float32 array
func stringToFloat32Array(s string) ([]float32, error) {
	if s == "" || s == "<nil>" || s == "NULL" {
		return nil, nil
	}
	if len(s) < 2 || s[0] != '[' || s[len(s)-1] != ']' {
		return nil, fmt.Errorf("invalid vector format: %s", s)
	}
	s = s[1 : len(s)-1]
	if s == "" {
		return []float32{}, nil
	}
	parts := strings.Split(s, ",")
	res := make([]float32, len(parts))
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 32)
		if err != nil {
			return nil, err
		}
		res[i] = float32(f)
	}
	return res, nil
}

func (r *memoryRepository) Create(ctx context.Context, memory *domain.Memory) error {
	query := `
		INSERT INTO memories (project_id, room_id, external_user_id, content, embedding, metadata)
		VALUES ($1, $2, $3, $4, $5::vector, $6)
		RETURNING id, created_at
	`
	embeddingStr := float32ArrayToString(memory.Embedding)
	
	type Inserted struct {
		ID        string `db:"id"`
		CreatedAt string `db:"created_at"`
	}
	var ins Inserted
	err := r.db.GetContext(ctx, &ins, query, 
		memory.ProjectID, 
		memory.RoomID, 
		memory.ExternalUserID, 
		memory.Content, 
		embeddingStr, 
		memory.Metadata,
	)
	if err != nil {
		return err
	}

	memory.ID = ins.ID
	return nil
}

func (r *memoryRepository) SearchSemantic(ctx context.Context, roomID string, externalUserID *string, queryEmbedding []float32, freshnessWeight float64, limit int) ([]domain.Memory, error) {
	embeddingStr := float32ArrayToString(queryEmbedding)
	semanticWeight := 1.0 - freshnessWeight

	var query string
	var args []any

	if externalUserID != nil && *externalUserID != "" {
		query = `
			SELECT id, project_id, room_id, external_user_id, content, embedding::text, metadata, created_at, status, superseded_by_id, shannon_entropy, is_whitelisted
			FROM memories
			WHERE (room_id = $1 OR external_user_id = $2) AND status = 'active'
			ORDER BY ($4 * (1 - (embedding <=> $3::vector)) + $5 * (1.0 / (1 + EXTRACT(EPOCH FROM (NOW() - created_at)) / 86400))) DESC
			LIMIT $6
		`
		args = []any{roomID, *externalUserID, embeddingStr, semanticWeight, freshnessWeight, limit}
	} else {
		query = `
			SELECT id, project_id, room_id, external_user_id, content, embedding::text, metadata, created_at, status, superseded_by_id, shannon_entropy, is_whitelisted
			FROM memories
			WHERE room_id = $1 AND status = 'active'
			ORDER BY ($3 * (1 - (embedding <=> $2::vector)) + $4 * (1.0 / (1 + EXTRACT(EPOCH FROM (NOW() - created_at)) / 86400))) DESC
			LIMIT $5
		`
		args = []any{roomID, embeddingStr, semanticWeight, freshnessWeight, limit}
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
	}

	var rows []dbMemory
	err := r.db.SelectContext(ctx, &rows, query, args...)
	if err != nil {
		return nil, err
	}

	memories := make([]domain.Memory, len(rows))
	for i, row := range rows {
		emb, err := stringToFloat32Array(row.EmbeddingText)
		if err != nil {
			return nil, err
		}
		memories[i] = domain.Memory{
			ID:             row.ID,
			ProjectID:      row.ProjectID,
			RoomID:         row.RoomID,
			ExternalUserID: row.ExternalUserID,
			Content:        row.Content,
			Embedding:      emb,
			Metadata:       row.Metadata,
		}
	}

	return memories, nil
}

func (r *memoryRepository) ListByRoom(ctx context.Context, roomID string) ([]domain.Memory, error) {
	query := `
		SELECT id, project_id, room_id, external_user_id, content, embedding::text, metadata, created_at
		FROM memories
		WHERE room_id = $1
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
	}

	var rows []dbMemory
	err := r.db.SelectContext(ctx, &rows, query, roomID)
	if err != nil {
		return nil, err
	}

	memories := make([]domain.Memory, len(rows))
	for i, row := range rows {
		emb, err := stringToFloat32Array(row.EmbeddingText)
		if err != nil {
			return nil, err
		}
		memories[i] = domain.Memory{
			ID:             row.ID,
			ProjectID:      row.ProjectID,
			RoomID:         row.RoomID,
			ExternalUserID: row.ExternalUserID,
			Content:        row.Content,
			Embedding:      emb,
			Metadata:       row.Metadata,
		}
	}

	return memories, nil
}
