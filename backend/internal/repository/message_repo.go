package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

type messageRepository struct {
	db       *sqlx.DB
	d1Store  domain.D1Store
	d1Router domain.D1Router
}

func NewMessageRepository(db *sqlx.DB, d1Store domain.D1Store, d1Router domain.D1Router) domain.MessageRepository {
	return &messageRepository{db: db, d1Store: d1Store, d1Router: d1Router}
}

// Helper to extract projectID from a room since messages route via project_id
func (r *messageRepository) getProjectID(ctx context.Context, roomID string) (string, error) {
	var projectID string
	err := r.db.GetContext(ctx, &projectID, "SELECT project_id FROM rooms WHERE id = $1", roomID)
	return projectID, err
}

func (r *messageRepository) Create(ctx context.Context, msg *domain.Message) error {
	projectID, err := r.getProjectID(ctx, msg.RoomID)
	if err != nil {
		return fmt.Errorf("failed to get project ID for room %s: %w", msg.RoomID, err)
	}

	d1UUID, err := r.d1Router.Lookup(ctx, projectID)
	if err != nil {
		return err
	}

	// 1. Get max sequence
	maxSeqResults, err := r.d1Store.Query(ctx, d1UUID, "SELECT COALESCE(MAX(sequence), 0) as max_seq FROM messages WHERE room_id = ?", msg.RoomID)
	if err != nil {
		return err
	}
	
	var maxSeq int64 = 0
	if len(maxSeqResults) > 0 {
		if val, ok := maxSeqResults[0]["max_seq"].(float64); ok {
			maxSeq = int64(val)
		}
	}
	msg.Sequence = maxSeq + 1

	// 2. Insert message
	query := `
		INSERT INTO messages (id, room_id, role, content, sequence, created_at, tokens_input, tokens_output, cost_total, is_encrypted, encryption_metadata)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	createdAt := msg.CreatedAt
	if createdAt == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339)
	}

	_, err = r.d1Store.Query(ctx, d1UUID, query, msg.ID, msg.RoomID, msg.Role, msg.Content, msg.Sequence, createdAt, msg.TokensInput, msg.TokensOutput, msg.CostTotal, msg.IsEncrypted, msg.EncryptionMetadata)
	
	if err == nil {
		go func() {
			bgCtx := context.Background()
			updateQuery := `
				UPDATE rooms 
				SET message_count = message_count + 1, last_message_at = $1 
				WHERE id = $2
			`
			_, err := r.db.ExecContext(bgCtx, updateQuery, createdAt, msg.RoomID)
			if err != nil {
				fmt.Printf("[Message Repo Background Error] Failed to update room message metrics: %v\n", err)
			}
		}()
	}

	return err
}

func (r *messageRepository) ListByRoom(ctx context.Context, roomID string, limit int, beforeSeq int64) ([]domain.Message, error) {
	projectID, err := r.getProjectID(ctx, roomID)
	if err != nil {
		return nil, err
	}

	d1UUID, err := r.d1Router.Lookup(ctx, projectID)
	if err != nil {
		return nil, err
	}

	var query string
	var args []any
	if beforeSeq > 0 {
		query = `
			SELECT id, room_id, role, content, sequence, created_at, tokens_input, tokens_output, cost_total, COALESCE(is_encrypted, false) AS is_encrypted, encryption_metadata
			FROM messages
			WHERE room_id = ? AND sequence < ?
			ORDER BY sequence DESC
			LIMIT ?
		`
		args = []any{roomID, beforeSeq, limit}
	} else {
		query = `
			SELECT id, room_id, role, content, sequence, created_at, tokens_input, tokens_output, cost_total, COALESCE(is_encrypted, false) AS is_encrypted, encryption_metadata
			FROM messages
			WHERE room_id = ?
			ORDER BY sequence DESC
			LIMIT ?
		`
		args = []any{roomID, limit}
	}

	results, err := r.d1Store.Query(ctx, d1UUID, query, args...)
	if err != nil {
		return nil, err
	}

	var messages []domain.Message
	for _, row := range results {
		msg := domain.Message{
			ID:      row["id"].(string),
			RoomID:  row["room_id"].(string),
			Role:    row["role"].(string),
			Content: row["content"].(string),
		}
		if seq, ok := row["sequence"].(float64); ok {
			msg.Sequence = int64(seq)
		}
		if createdAt, ok := row["created_at"].(string); ok {
			msg.CreatedAt = createdAt
		}
		// Type asserting all possible types correctly in Go is verbose for maps, but this is the MVP parsing
		messages = append(messages, msg)
	}

	// Reverse slice to return in chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (r *messageRepository) GetMaxSequence(ctx context.Context, roomID string) (int64, error) {
	projectID, err := r.getProjectID(ctx, roomID)
	if err != nil {
		return 0, err
	}

	d1UUID, err := r.d1Router.Lookup(ctx, projectID)
	if err != nil {
		return 0, err
	}

	results, err := r.d1Store.Query(ctx, d1UUID, "SELECT COALESCE(MAX(sequence), 0) as max_seq FROM messages WHERE room_id = ?", roomID)
	if err != nil || len(results) == 0 {
		return 0, err
	}

	if val, ok := results[0]["max_seq"].(float64); ok {
		return int64(val), nil
	}
	return 0, nil
}

func (r *messageRepository) ListProjectLogs(ctx context.Context, projectID string, roomID string, externalUserID string, search string, role string, sortBy string, sortOrder string, limit int, offset int) ([]domain.ProjectLogEntry, int, error) {
	allowedSorts := map[string]string{
		"created_at": "m.created_at",
		"cost_total": "COALESCE(m.cost_total, 0)",
		"tokens":     "(COALESCE(m.tokens_input, 0) + COALESCE(m.tokens_output, 0))",
	}

	orderBy := "m.created_at"
	if s, ok := allowedSorts[sortBy]; ok {
		orderBy = s
	}

	dir := "DESC"
	if strings.ToUpper(sortOrder) == "ASC" {
		dir = "ASC"
	}

	queryConds := "WHERE r.project_id = $1"
	args := []any{projectID}
	paramIdx := 2

	if roomID != "" {
		queryConds += fmt.Sprintf(" AND m.room_id::text ILIKE $%d", paramIdx)
		args = append(args, "%"+roomID+"%")
		paramIdx++
	}

	if externalUserID != "" {
		queryConds += fmt.Sprintf(" AND r.external_user_id ILIKE $%d", paramIdx)
		args = append(args, "%"+externalUserID+"%")
		paramIdx++
	}

	if role != "" {
		queryConds += fmt.Sprintf(" AND m.role = $%d", paramIdx)
		args = append(args, role)
		paramIdx++
	}

	if search != "" {
		queryConds += fmt.Sprintf(" AND (m.content ILIKE $%d OR m.room_id::text ILIKE $%d OR r.external_user_id ILIKE $%d)", paramIdx, paramIdx, paramIdx)
		args = append(args, "%"+search+"%")
		paramIdx++
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) 
		FROM messages m
		JOIN rooms r ON r.id = m.room_id
		%s
	`, queryConds)

	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	selectQuery := fmt.Sprintf(`
		SELECT 
			m.id,
			m.room_id,
			COALESCE(r.external_user_id, '') AS external_user_id,
			m.role,
			m.content,
			m.sequence,
			m.created_at,
			m.tokens_input,
			m.tokens_output,
			m.cost_total,
			COALESCE(m.is_encrypted, false) AS is_encrypted,
			m.encryption_metadata
		FROM messages m
		JOIN rooms r ON r.id = m.room_id
		%s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, queryConds, orderBy, dir, paramIdx, paramIdx+1)

	args = append(args, limit, offset)

	var logs []domain.ProjectLogEntry
	if err := r.db.SelectContext(ctx, &logs, selectQuery, args...); err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
