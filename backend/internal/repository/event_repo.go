package repository

import (
	"context"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

type eventRepository struct {
	db *sqlx.DB
}

func NewEventRepository(db *sqlx.DB) domain.EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) Create(ctx context.Context, event *domain.RoomEvent) error {
	query := `
		INSERT INTO room_events (room_id, event_type, payload)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	type Inserted struct {
		ID        string `db:"id"`
		CreatedAt string `db:"created_at"`
	}
	var ins Inserted
	err := r.db.GetContext(ctx, &ins, query, event.RoomID, event.EventType, event.Payload)
	if err != nil {
		return err
	}

	event.ID = ins.ID
	event.CreatedAt = ins.CreatedAt
	return nil
}

func (r *eventRepository) ListByRoom(ctx context.Context, roomID string, limit int) ([]domain.RoomEvent, error) {
	query := `
		SELECT id, room_id, event_type, payload, created_at
		FROM room_events
		WHERE room_id = $1
		ORDER BY created_at DESC
		LIMIT $2
	`
	var events []domain.RoomEvent
	err := r.db.SelectContext(ctx, &events, query, roomID, limit)
	if err != nil {
		return nil, err
	}
	return events, nil
}
