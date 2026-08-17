package repository

import (
	"context"
	"database/sql"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

// MCPMemoryRepository executes memory queries and pgvector searches for the
// MCP server. Embeddings are stored directly in PostgreSQL via the vector
// extension, mirroring the pgvector vector store implementation.
type MCPMemoryRepository struct {
	db *sqlx.DB
}

func NewMCPMemoryRepository(db *sqlx.DB) *MCPMemoryRepository {
	return &MCPMemoryRepository{db: db}
}

// Create persists a memory. The embedding may be NULL (embedding is generated
// asynchronously by the MCP embedding worker). If embedding is provided it is
// written directly to the vector column.
func (r *MCPMemoryRepository) Create(ctx context.Context, memory *domain.Memory) error {
	query := `
		INSERT INTO memories (project_id, room_id, external_user_id, content, embedding, metadata)
		VALUES ($1, $2, $3, $4, $5::vector, $6)
		RETURNING id, created_at
	`
	var embeddingParam any
	if len(memory.Embedding) > 0 {
		embeddingParam = float32ArrayToString(memory.Embedding)
	}

	type Inserted struct {
		ID        string `db:"id"`
		CreatedAt string `db:"created_at"`
	}
	var ins Inserted
	if err := r.db.GetContext(ctx, &ins, query,
		memory.ProjectID,
		memory.RoomID,
		memory.ExternalUserID,
		memory.Content,
		embeddingParam,
		memory.Metadata,
	); err != nil {
		return err
	}
	memory.ID = ins.ID
	return nil
}

// UpdateEmbedding writes an asynchronously generated embedding onto a memory.
func (r *MCPMemoryRepository) UpdateEmbedding(ctx context.Context, memoryID string, embedding []float32) error {
	query := "UPDATE memories SET embedding = $2::vector WHERE id = $1"
	_, err := r.db.ExecContext(ctx, query, memoryID, float32ArrayToString(embedding))
	return err
}

// Search performs pidvector cosine similarity search scoped to the project and
// room, blending semantic similarity with freshness. Only active memories with
// a non-null embedding are considered.
func (r *MCPMemoryRepository) Search(ctx context.Context, projectID string, roomID string, externalUserID *string, queryEmbedding []float32, freshnessWeight float64, limit int) ([]domain.Memory, error) {
	if len(queryEmbedding) == 0 {
		return []domain.Memory{}, nil
	}
	embeddingStr := float32ArrayToString(queryEmbedding)
	semanticWeight := 1.0 - freshnessWeight

	const query = `
		SELECT id, project_id, room_id, external_user_id, content, metadata, created_at, status, superseded_by_id, shannon_entropy, is_whitelisted
		FROM memories
		WHERE project_id = $1 AND room_id = $2 AND status = 'active' AND embedding IS NOT NULL AND $3::vector IS NOT NULL
		ORDER BY ($5 * (1 - (embedding <=> $3::vector)) + $6 * (1.0 / (1 + EXTRACT(EPOCH FROM (NOW() - created_at)) / 86400))) DESC
		LIMIT $4
	`

	var rows []mcpDbMemory
	if err := r.db.SelectContext(ctx, &rows, query, projectID, roomID, embeddingStr, limit, semanticWeight, freshnessWeight); err != nil {
		return nil, err
	}

	memories := make([]domain.Memory, len(rows))
	for i, row := range rows {
		memories[i] = row.toMemory()
	}
	return memories, nil
}

// ListByRoom returns all memories for a room with pagination, seeded by blank
// (zero) scores; the pgvector score is unavailable in list mode.
func (r *MCPMemoryRepository) ListByRoom(ctx context.Context, projectID string, roomID string, limit int, offset int) ([]domain.Memory, int, error) {
	countQuery := `
		SELECT COUNT(*) FROM memories WHERE project_id = $1 AND room_id = $2
	`
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, projectID, roomID); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, project_id, room_id, external_user_id, content, metadata, created_at, status, superseded_by_id, shannon_entropy, is_whitelisted
		FROM memories
		WHERE project_id = $1 AND room_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`
	var rows []mcpDbMemory
	if err := r.db.SelectContext(ctx, &rows, query, projectID, roomID, limit, offset); err != nil {
		return nil, 0, err
	}

	memories := make([]domain.Memory, len(rows))
	for i, row := range rows {
		memories[i] = row.toMemory()
	}
	return memories, total, nil
}

// Delete removes a memory row, enforcing project scope.
func (r *MCPMemoryRepository) Delete(ctx context.Context, projectID string, memoryID string) error {
	query := `
		DELETE FROM memories
		WHERE id = $1 AND project_id = $2
	`
	res, err := r.db.ExecContext(ctx, query, memoryID, projectID)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	return nil
}

type mcpDbMemory struct {
	ID             string         `db:"id"`
	ProjectID      string         `db:"project_id"`
	RoomID         *string        `db:"room_id"`
	ExternalUserID *string        `db:"external_user_id"`
	Content        string         `db:"content"`
	Metadata       string         `db:"metadata"`
	CreatedAt      string         `db:"created_at"`
	Status         string         `db:"status"`
	SupersededByID *string        `db:"superseded_by_id"`
	ShannonEntropy float64        `db:"shannon_entropy"`
	IsWhitelisted  bool           `db:"is_whitelisted"`
}

func (row mcpDbMemory) toMemory() domain.Memory {
	return domain.Memory{
		ID:             row.ID,
		ProjectID:      row.ProjectID,
		RoomID:         row.RoomID,
		ExternalUserID: row.ExternalUserID,
		Content:        row.Content,
		Metadata:       row.Metadata,
		Status:         row.Status,
		SupersededByID: row.SupersededByID,
		ShannonEntropy: row.ShannonEntropy,
		IsWhitelisted:  row.IsWhitelisted,
	}
}