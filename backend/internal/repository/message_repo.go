package repository

import (
	"context"
	"database/sql"
	"errors"

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

	// Get next sequence value
	var maxSeq int64
	err = tx.GetContext(ctx, &maxSeq, "SELECT COALESCE(MAX(sequence), 0) FROM messages WHERE room_id = $1", msg.RoomID)
	if err != nil {
		return err
	}

	msg.Sequence = maxSeq + 1

	query := `
		INSERT INTO messages (room_id, role, content, sequence, tokens_input, tokens_output, cost_total)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at
	`
	// To scan fields from RETURNING
	type Inserted struct {
		ID        string `db:"id"`
		CreatedAt string `db:"created_at"`
	}
	var ins Inserted
	err = tx.GetContext(ctx, &ins, query, msg.RoomID, msg.Role, msg.Content, msg.Sequence, msg.TokensInput, msg.TokensOutput, msg.CostTotal)
	if err != nil {
		return err
	}

	msg.ID = ins.ID
	msg.CreatedAt = ins.CreatedAt

	return tx.Commit()
}

func (r *messageRepository) ListByRoom(ctx context.Context, roomID string, limit int, beforeSeq int64) ([]domain.Message, error) {
	var query string
	var args []any
	if beforeSeq > 0 {
		query = `
			SELECT id, room_id, role, content, sequence, created_at, tokens_input, tokens_output, cost_total
			FROM messages
			WHERE room_id = $1 AND sequence < $2
			ORDER BY sequence DESC
			LIMIT $3
		`
		args = []any{roomID, beforeSeq, limit}
	} else {
		query = `
			SELECT id, room_id, role, content, sequence, created_at, tokens_input, tokens_output, cost_total
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
