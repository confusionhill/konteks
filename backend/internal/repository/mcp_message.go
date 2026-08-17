package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

// MCPMessageRepository executes message queries for the MCP server.
type MCPMessageRepository struct {
	db *sqlx.DB
}

func NewMCPMessageRepository(db *sqlx.DB) *MCPMessageRepository {
	return &MCPMessageRepository{db: db}
}

func (r *MCPMessageRepository) Create(ctx context.Context, msg *domain.Message) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var maxSeq int64
	if err := tx.GetContext(ctx, &maxSeq, "SELECT COALESCE(MAX(sequence), 0) FROM messages WHERE room_id = $1", msg.RoomID); err != nil {
		return err
	}
	msg.Sequence = maxSeq + 1

	query := `
		INSERT INTO messages (id, room_id, role, content, sequence, tokens_input, tokens_output, cost_total, is_encrypted, encryption_metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING created_at
	`
	if err := tx.GetContext(ctx, &msg.CreatedAt, query, msg.ID, msg.RoomID, msg.Role, msg.Content, msg.Sequence, msg.TokensInput, msg.TokensOutput, msg.CostTotal, msg.IsEncrypted, msg.EncryptionMetadata); err != nil {
		return err
	}
	return tx.Commit()
}

// ListByRoom returns messages in chronological order, newest first within the
// pagination window when beforeSeq is 0.
func (r *MCPMessageRepository) ListByRoom(ctx context.Context, roomID string, limit int, beforeSeq int64) ([]domain.Message, error) {
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
	if err := r.db.SelectContext(ctx, &messages, query, args...); err != nil {
		return nil, err
	}
	if messages == nil {
		messages = []domain.Message{}
	}
	// Reverse slice to return in chronological order
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}
	return messages, nil
}

// ListByRoomRange returns a chronological window of messages.
func (r *MCPMessageRepository) ListByRoomRange(ctx context.Context, roomID string, limit int, offset int) ([]domain.Message, error) {
	query := `
		SELECT id, room_id, role, content, sequence, created_at, tokens_input, tokens_output, cost_total, COALESCE(is_encrypted, false) AS is_encrypted, encryption_metadata
		FROM messages
		WHERE room_id = $1
		ORDER BY sequence ASC
		LIMIT $2 OFFSET $3
	`
	var messages []domain.Message
	if err := r.db.SelectContext(ctx, &messages, query, roomID, limit, offset); err != nil {
		return nil, err
	}
	if messages == nil {
		messages = []domain.Message{}
	}
	return messages, nil
}

func (r *MCPMessageRepository) CountByRoom(ctx context.Context, roomID string) (int, error) {
	var total int
	if err := r.db.GetContext(ctx, &total, "SELECT COUNT(*) FROM messages WHERE room_id = $1", roomID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return total, nil
}

func (r *MCPMessageRepository) GetMaxSequence(ctx context.Context, roomID string) (int64, error) {
	var maxSeq int64
	if err := r.db.GetContext(ctx, &maxSeq, "SELECT COALESCE(MAX(sequence), 0) FROM messages WHERE room_id = $1", roomID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, nil
		}
		return 0, err
	}
	return maxSeq, nil
}