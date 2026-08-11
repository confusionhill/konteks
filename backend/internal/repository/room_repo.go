package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

type roomRepository struct {
	db *sqlx.DB
}

func NewRoomRepository(db *sqlx.DB) domain.RoomRepository {
	return &roomRepository{db: db}
}

func (r *roomRepository) Create(ctx context.Context, tenantID string, externalUserID string) (*domain.Room, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Insert room
	var room domain.Room
	queryRoom := `
		INSERT INTO rooms (tenant_id, external_user_id)
		VALUES ($1, $2)
		RETURNING id, tenant_id, external_user_id, created_at, updated_at
	`
	err = tx.GetContext(ctx, &room, queryRoom, tenantID, externalUserID)
	if err != nil {
		return nil, err
	}

	// Insert initial empty state
	queryState := `
		INSERT INTO room_state (room_id, summary, active_topic, state)
		VALUES ($1, '', '', '{}'::jsonb)
	`
	_, err = tx.ExecContext(ctx, queryState, room.ID)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &room, nil
}

func (r *roomRepository) GetByID(ctx context.Context, tenantID string, roomID string) (*domain.Room, error) {
	query := `
		SELECT id, tenant_id, external_user_id, created_at, updated_at
		FROM rooms
		WHERE tenant_id = $1 AND id = $2
	`
	var room domain.Room
	err := r.db.GetContext(ctx, &room, query, tenantID, roomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	return &room, nil
}

func (r *roomRepository) Delete(ctx context.Context, tenantID string, roomID string) error {
	// First verify room belongs to tenant
	var count int
	err := r.db.GetContext(ctx, &count, "SELECT COUNT(*) FROM rooms WHERE tenant_id = $1 AND id = $2", tenantID, roomID)
	if err != nil {
		return err
	}
	if count == 0 {
		return nil // Nothing to delete / unauthorized
	}

	query := "DELETE FROM rooms WHERE id = $1"
	_, err = r.db.ExecContext(ctx, query, roomID)
	return err
}

func (r *roomRepository) GetState(ctx context.Context, roomID string) (*domain.RoomState, error) {
	query := `
		SELECT room_id, summary, active_topic, state, updated_at
		FROM room_state
		WHERE room_id = $1
	`
	var state domain.RoomState
	err := r.db.GetContext(ctx, &state, query, roomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	return &state, nil
}

func (r *roomRepository) UpsertState(ctx context.Context, state *domain.RoomState) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	queryState := `
		INSERT INTO room_state (room_id, summary, active_topic, state, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (room_id) DO UPDATE SET
			summary = EXCLUDED.summary,
			active_topic = EXCLUDED.active_topic,
			state = EXCLUDED.state,
			updated_at = EXCLUDED.updated_at
	`
	now := time.Now()
	_, err = tx.ExecContext(ctx, queryState, state.RoomID, state.Summary, state.ActiveTopic, state.State, now)
	if err != nil {
		return err
	}

	queryRoomUpdate := "UPDATE rooms SET updated_at = $1 WHERE id = $2"
	_, err = tx.ExecContext(ctx, queryRoomUpdate, now, state.RoomID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
