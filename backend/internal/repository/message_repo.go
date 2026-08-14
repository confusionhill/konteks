package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

type messageRepository struct {
	db *sqlx.DB
}

func NewMessageRepository(db *sqlx.DB) domain.MessageRepository {
	return &messageRepository{db: db}
}

func (r *messageRepository) Create(ctx context.Context, msg *domain.Message) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var maxSeq int64
	err = tx.GetContext(ctx, &maxSeq, "SELECT COALESCE(MAX(sequence), 0) FROM messages WHERE room_id = $1", msg.RoomID)
	if err != nil {
		return err
	}

	msg.Sequence = maxSeq + 1

	query := `
		INSERT INTO messages (id, room_id, role, content, sequence, tokens_input, tokens_output, cost_total, is_encrypted, encryption_metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err = tx.ExecContext(ctx, query, msg.ID, msg.RoomID, msg.Role, msg.Content, msg.Sequence, msg.TokensInput, msg.TokensOutput, msg.CostTotal, msg.IsEncrypted, msg.EncryptionMetadata)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *messageRepository) ListByRoom(ctx context.Context, roomID string, limit int, beforeSeq int64) ([]domain.Message, error) {
	var query string
	var args []any
	if beforeSeq > 0 {
		query = `
			SELECT id, room_id, role, content, sequence, created_at, tokens_input, tokens_output, cost_total, COALESCE(is_encrypted, false) AS is_encrypted, encryption_metadata
			FROM messages
			WHERE room_id = $1 AND sequence < $2
			ORDER BY sequence DESC
			LIMIT $3
		`
		args = []any{roomID, beforeSeq, limit}
	} else {
		query = `
			SELECT id, room_id, role, content, sequence, created_at, tokens_input, tokens_output, cost_total, COALESCE(is_encrypted, false) AS is_encrypted, encryption_metadata
			FROM messages
			WHERE room_id = $1
			ORDER BY sequence DESC
			LIMIT $2
		`
		args = []any{roomID, limit}
	}

	var messages []domain.Message
	err := r.db.SelectContext(ctx, &messages, query, args...)
	if err != nil {
		return nil, err
	}

	// Reverse slice to return in chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, nil
}

func (r *messageRepository) GetMaxSequence(ctx context.Context, roomID string) (int64, error) {
	var maxSeq int64
	err := r.db.GetContext(ctx, &maxSeq, "SELECT COALESCE(MAX(sequence), 0) FROM messages WHERE room_id = $1", roomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return maxSeq, nil
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
