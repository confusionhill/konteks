package repository

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

// memorySearchOverfetch is the factor by which we pull extra candidates from
// the vector store so a freshness re-rank in Go keeps enough results after
// superseded/stale memories are filtered out.
const memorySearchOverfetch = 4

type memoryRepository struct {
	db          *sqlx.DB
	vectorStore domain.VectorStore
}

func NewMemoryRepository(db *sqlx.DB, vectorStore domain.VectorStore) domain.MemoryRepository {
	return &memoryRepository{db: db, vectorStore: vectorStore}
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
		INSERT INTO memories (project_id, room_id, external_user_id, content, metadata, status)
		VALUES ($1, $2, $3, $4, $5, 'active')
		RETURNING id, created_at
	`

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
		memory.Metadata,
	)
	if err != nil {
		return err
	}

	memory.ID = ins.ID

	if r.vectorStore != nil {
		if errVec := r.vectorStore.UpsertMemoryVector(ctx, memory); errVec != nil {
			// Log error but do not fail DB record creation
			fmt.Printf("[MemoryRepository Error]: failed to upsert vector: %v\n", errVec)
		}
	}
	return nil
}

func (r *memoryRepository) SearchSemantic(ctx context.Context, projectID string, roomID string, externalUserID *string, queryEmbedding []float32, freshnessWeight float64, limit int) ([]domain.Memory, error) {
	if len(queryEmbedding) == 0 {
		return []domain.Memory{}, nil
	}
	if r.vectorStore == nil {
		return nil, fmt.Errorf("vector store not configured")
	}

	if limit <= 0 {
		limit = 5
	}

	matches, err := r.vectorStore.SearchMemoryVectors(ctx, projectID, roomID, externalUserID, queryEmbedding, limit*memorySearchOverfetch)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
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
		createdAt := row.CreatedAt
		secs := now.Sub(createdAt).Seconds()
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

type memoryRow struct {
	Memory    domain.Memory
	CreatedAt time.Time
}

func (r *memoryRepository) getByIDs(ctx context.Context, ids []string) (map[string]memoryRow, error) {
	type dbMemory struct {
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

	query := `
		SELECT id, project_id, room_id, external_user_id, content, metadata, created_at, status, superseded_by_id, shannon_entropy, is_whitelisted
		FROM memories
		WHERE id = ANY($1::uuid[])
	`
	var rows []dbMemory
	if err := r.db.SelectContext(ctx, &rows, query, ids); err != nil {
		return nil, err
	}

	result := make(map[string]memoryRow, len(rows))
	for _, row := range rows {
		result[row.ID] = memoryRow{
			Memory: domain.Memory{
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
			},
			CreatedAt: row.CreatedAt,
		}
	}
	return result, nil
}

func (r *memoryRepository) ListByRoom(ctx context.Context, roomID string) ([]domain.Memory, error) {
	query := `
		SELECT id, project_id, room_id, external_user_id, content, metadata, created_at, status, superseded_by_id, shannon_entropy, is_whitelisted
		FROM memories
		WHERE room_id = $1
	`
	type dbMemory struct {
		ID             string  `db:"id"`
		ProjectID      string  `db:"project_id"`
		RoomID         *string `db:"room_id"`
		ExternalUserID *string `db:"external_user_id"`
		Content        string  `db:"content"`
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
		return nil, err
	}

	memories := make([]domain.Memory, len(rows))
	for i, row := range rows {
		memories[i] = domain.Memory{
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

	return memories, nil
}
