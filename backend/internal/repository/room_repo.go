package repository

import (
	"context"
	"database/sql"
	"encoding/json"
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

func (r *roomRepository) Create(ctx context.Context, tenantID string, externalUserID string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy) (*domain.Room, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Insert room
	var room domain.Room
	queryRoom := `
		INSERT INTO rooms (tenant_id, external_user_id, state_schema, default_policy)
		VALUES ($1, $2, $3, $4)
		RETURNING id, tenant_id, external_user_id, created_at, updated_at, state_schema, default_policy
	`
	var schemaParam any = nil
	if len(stateSchema) > 0 && string(stateSchema) != "null" {
		schemaParam = string(stateSchema)
	}
	var policyParam any = nil
	if defaultPolicy != nil {
		pBytes, _ := json.Marshal(defaultPolicy)
		policyParam = string(pBytes)
	}

	type rawRoom struct {
		ID             string         `db:"id"`
		TenantID       string         `db:"tenant_id"`
		ExternalUserID string         `db:"external_user_id"`
		CreatedAt      string         `db:"created_at"`
		UpdatedAt      string         `db:"updated_at"`
		StateSchema    sql.NullString `db:"state_schema"`
		DefaultPolicy  sql.NullString `db:"default_policy"`
	}
	var rr rawRoom
	err = tx.GetContext(ctx, &rr, queryRoom, tenantID, externalUserID, schemaParam, policyParam)
	if err != nil {
		return nil, err
	}
	room = domain.Room{
		ID:             rr.ID,
		TenantID:       rr.TenantID,
		ExternalUserID: rr.ExternalUserID,
		CreatedAt:      rr.CreatedAt,
		UpdatedAt:      rr.UpdatedAt,
	}
	if rr.StateSchema.Valid {
		room.StateSchema = []byte(rr.StateSchema.String)
	}
	if rr.DefaultPolicy.Valid {
		var p domain.ContextPolicy
		_ = json.Unmarshal([]byte(rr.DefaultPolicy.String), &p)
		room.DefaultPolicy = &p
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
		SELECT id, tenant_id, external_user_id, created_at, updated_at, state_schema, default_policy
		FROM rooms
		WHERE tenant_id = $1 AND id = $2
	`
	type rawRoom struct {
		ID             string         `db:"id"`
		TenantID       string         `db:"tenant_id"`
		ExternalUserID string         `db:"external_user_id"`
		CreatedAt      string         `db:"created_at"`
		UpdatedAt      string         `db:"updated_at"`
		StateSchema    sql.NullString `db:"state_schema"`
		DefaultPolicy  sql.NullString `db:"default_policy"`
	}
	var rr rawRoom
	err := r.db.GetContext(ctx, &rr, query, tenantID, roomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}
	room := domain.Room{
		ID:             rr.ID,
		TenantID:       rr.TenantID,
		ExternalUserID: rr.ExternalUserID,
		CreatedAt:      rr.CreatedAt,
		UpdatedAt:      rr.UpdatedAt,
	}
	if rr.StateSchema.Valid && rr.StateSchema.String != "" {
		room.StateSchema = []byte(rr.StateSchema.String)
	}
	if rr.DefaultPolicy.Valid && rr.DefaultPolicy.String != "" {
		var p domain.ContextPolicy
		_ = json.Unmarshal([]byte(rr.DefaultPolicy.String), &p)
		room.DefaultPolicy = &p
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
	type rawRoomState struct {
		RoomID      string         `db:"room_id"`
		Summary     string         `db:"summary"`
		ActiveTopic string         `db:"active_topic"`
		State       sql.NullString `db:"state"`
		References  sql.NullString `db:"references"`
		Entities    sql.NullString `db:"entities"`
		UpdatedAt   string         `db:"updated_at"`
	}
	query := `
		SELECT room_id, summary, active_topic, state, "references", "entities", updated_at
		FROM room_state
		WHERE room_id = $1
	`
	var raw rawRoomState
	err := r.db.GetContext(ctx, &raw, query, roomID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil // Not found
		}
		return nil, err
	}

	state := &domain.RoomState{
		RoomID:      raw.RoomID,
		Summary:     raw.Summary,
		ActiveTopic: raw.ActiveTopic,
		State:       []byte(raw.State.String),
		UpdatedAt:   raw.UpdatedAt,
	}

	if raw.References.Valid && raw.References.String != "" {
		_ = json.Unmarshal([]byte(raw.References.String), &state.References)
	}
	if raw.Entities.Valid && raw.Entities.String != "" {
		_ = json.Unmarshal([]byte(raw.Entities.String), &state.Entities)
	}

	return state, nil
}

func (r *roomRepository) UpsertState(ctx context.Context, state *domain.RoomState) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	refBytes, _ := json.Marshal(state.References)
	if state.References == nil {
		refBytes = []byte("{}")
	}
	entBytes, _ := json.Marshal(state.Entities)
	if state.Entities == nil {
		entBytes = []byte("[]")
	}

	queryState := `
		INSERT INTO room_state (room_id, summary, active_topic, state, "references", "entities", updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (room_id) DO UPDATE SET
			summary = EXCLUDED.summary,
			active_topic = EXCLUDED.active_topic,
			state = EXCLUDED.state,
			"references" = EXCLUDED."references",
			"entities" = EXCLUDED."entities",
			updated_at = EXCLUDED.updated_at
	`
	now := time.Now()
	_, err = tx.ExecContext(ctx, queryState, state.RoomID, state.Summary, state.ActiveTopic, state.State, string(refBytes), string(entBytes), now)
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

func (r *roomRepository) ListWithOverview(ctx context.Context, tenantID string) ([]domain.RoomOverview, error) {
	query := `
		SELECT
			r.id,
			COALESCE(r.external_user_id, '') AS external_user_id,
			COALESCE(rs.active_topic, '') AS active_topic,
			COALESCE(mc.msg_count, 0) AS message_count,
			COALESCE(uc.total_tokens, 0) AS total_tokens_used,
			COALESCE(uc.total_cost, 0) AS total_cost,
			COALESCE(uc.avg_tokens, 0) AS avg_tokens_per_turn,
			r.updated_at AS last_active_at
		FROM rooms r
		LEFT JOIN room_state rs ON rs.room_id = r.id
		LEFT JOIN (
			SELECT room_id, COUNT(*) AS msg_count FROM messages GROUP BY room_id
		) mc ON mc.room_id = r.id
		LEFT JOIN (
			SELECT 
				room_id, 
				SUM(tokens_used) AS total_tokens, 
				SUM(cost_total) AS total_cost, 
				AVG(tokens_used) AS avg_tokens 
			FROM usage_records 
			WHERE room_id IS NOT NULL 
			GROUP BY room_id
		) uc ON uc.room_id = r.id
		WHERE r.tenant_id = $1
		ORDER BY r.updated_at DESC
		LIMIT 50
	`
	var overviews []domain.RoomOverview
	err := r.db.SelectContext(ctx, &overviews, query, tenantID)
	if err != nil {
		return nil, err
	}
	return overviews, nil
}

