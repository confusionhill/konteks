package repository

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

// MCPMemoryRepository executes memory queries and Cloudflare Vectorize searches
// for the MCP server. Embeddings are stored only in Cloudflare Vectorize; the
// Postgres row carries the content and lifecycle metadata.
type MCPMemoryRepository struct {
	db          *sqlx.DB
	vectorStore domain.VectorStore
}

func NewMCPMemoryRepository(db *sqlx.DB, vectorStore domain.VectorStore) *MCPMemoryRepository {
	return &MCPMemoryRepository{db: db, vectorStore: vectorStore}
}

// Create persists a memory. The embedding is generated asynchronously by the
// MCP embedding worker and pushed to the vector store afterwards.
func (r *MCPMemoryRepository) Create(ctx context.Context, memory *domain.Memory) error {
	query := `
		INSERT INTO memories (project_id, room_id, external_user_id, content, metadata, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`
	var insID string
	if err := r.db.GetContext(ctx, &insID, query,
		memory.ProjectID,
		memory.RoomID,
		memory.ExternalUserID,
		memory.Content,
		memory.Metadata,
		memory.Status,
	); err != nil {
		return err
	}
	memory.ID = insID
	return nil
}

// UpdateEmbedding pushes an asynchronously generated embedding to the vector
// store. There is no Postgres vector column anymore.
func (r *MCPMemoryRepository) UpdateEmbedding(ctx context.Context, memory domain.Memory, embedding []float32) error {
	if r.vectorStore == nil {
		return nil
	}
	memory.Embedding = embedding
	if errVec := r.vectorStore.UpsertMemoryVector(ctx, &memory); errVec != nil {
		return fmt.Errorf("failed to upsert vector: %w", errVec)
	}
	return nil
}

// Search performs Cloudflare Vectorize cosine similarity search scoped to the
// project and room, hydrating matching rows from Postgres and blending
// semantic similarity with freshness in Go. Only active memories are returned.
func (r *MCPMemoryRepository) Search(ctx context.Context, projectID string, roomID string, externalUserID *string, queryEmbedding []float32, freshnessWeight float64, limit int) ([]domain.Memory, error) {
	if len(queryEmbedding) == 0 {
		return []domain.Memory{}, nil
	}
	if r.vectorStore == nil {
		return []domain.Memory{}, nil
	}
	if limit <= 0 {
		limit = 10
	}

	matches, err := r.vectorStore.SearchMemoryVectors(ctx, projectID, roomID, externalUserID, queryEmbedding, limit*4)
	if err != nil {
		return nil, err
	}
	if len(matches) == 0 {
		return []domain.Memory{}, nil
	}

	ids := make([]string, len(matches))
	for i, m := range matches {
		ids[i] = m.ID
	}

	rows, err := r.getByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	semanticWeight := 1.0 - freshnessWeight
	now := time.Now()

	type scoredMemory struct {
		mem   domain.Memory
		score float64
	}
	scored := make([]scoredMemory, 0, len(matches))
	for _, match := range matches {
		row, ok := rows[match.ID]
		if !ok {
			continue
		}
		if row.Memory.Status != "" && row.Memory.Status != domain.MemoryStatusActive {
			continue
		}
		secs := now.Sub(row.CreatedAt).Seconds()
		if secs < 0 {
			secs = 0
		}
		freshness := 1.0 / (1.0 + secs/86400.0)
		score := semanticWeight*match.Score + freshnessWeight*freshness
		scored = append(scored, scoredMemory{mem: row.Memory, score: score})
	}

	sort.Slice(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
	if len(scored) > limit {
		scored = scored[:limit]
	}

	memories := make([]domain.Memory, len(scored))
	for i, s := range scored {
		memories[i] = s.mem
	}
	return memories, nil
}

type mcpMemoryRow struct {
	Memory    domain.Memory
	CreatedAt time.Time
}

func (r *MCPMemoryRepository) getByIDs(ctx context.Context, ids []string) (map[string]mcpMemoryRow, error) {
	query := `
		SELECT id, project_id, room_id, external_user_id, content, metadata, created_at, status, superseded_by_id, shannon_entropy, is_whitelisted
		FROM memories
		WHERE id = ANY($1::uuid[])
	`
	var rows []mcpDbMemory
	if err := r.db.SelectContext(ctx, &rows, query, ids); err != nil {
		return nil, err
	}

	result := make(map[string]mcpMemoryRow, len(rows))
	for _, row := range rows {
		result[row.ID] = mcpMemoryRow{
			Memory:    row.toMemory(),
			CreatedAt: row.CreatedAt,
		}
	}
	return result, nil
}

// ListByRoom returns all memories for a room with pagination, seeded by blank
// (zero) scores; the vector score is unavailable in list mode.
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

// Delete removes a memory row and purges its vector from the store,
// enforcing project scope.
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

	if r.vectorStore != nil {
		if errVec := r.vectorStore.DeleteMemoryVectors(ctx, []string{memoryID}); errVec != nil {
			return fmt.Errorf("failed to purge vector: %w", errVec)
		}
	}
	return nil
}

type mcpDbMemory struct {
	ID             string    `db:"id"`
	ProjectID      string    `db:"project_id"`
	RoomID         *string   `db:"room_id"`
	ExternalUserID *string   `db:"external_user_id"`
	Content        string    `db:"content"`
	Metadata       string    `db:"metadata"`
	CreatedAt      time.Time `db:"created_at"`
	Status         string    `db:"status"`
	SupersededByID *string   `db:"superseded_by_id"`
	ShannonEntropy float64   `db:"shannon_entropy"`
	IsWhitelisted  bool      `db:"is_whitelisted"`
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