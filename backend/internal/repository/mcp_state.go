package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

// MCPStateRepository reads and patches room state for the MCP server.
type MCPStateRepository struct {
	db *sqlx.DB
}

func NewMCPStateRepository(db *sqlx.DB) *MCPStateRepository {
	return &MCPStateRepository{db: db}
}

func (r *MCPStateRepository) Get(ctx context.Context, roomID string) (*domain.RoomState, error) {
	query := `
		SELECT room_id, summary, active_topic, state, "references", "entities", last_summarized_seq, updated_at
		FROM room_state
		WHERE room_id = $1
	`
	type rawRoomState struct {
		RoomID            string         `db:"room_id"`
		Summary           string         `db:"summary"`
		ActiveTopic       string         `db:"active_topic"`
		State             sql.NullString `db:"state"`
		References        sql.NullString `db:"references"`
		Entities          sql.NullString `db:"entities"`
		LastSummarizedSeq int64          `db:"last_summarized_seq"`
		UpdatedAt         string         `db:"updated_at"`
	}
	var raw rawRoomState
	if err := r.db.GetContext(ctx, &raw, query, roomID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	state := &domain.RoomState{
		RoomID:            raw.RoomID,
		Summary:           raw.Summary,
		ActiveTopic:       raw.ActiveTopic,
		State:             []byte(raw.State.String),
		LastSummarizedSeq: raw.LastSummarizedSeq,
		UpdatedAt:         raw.UpdatedAt,
	}
	if raw.References.Valid && raw.References.String != "" {
		_ = json.Unmarshal([]byte(raw.References.String), &state.References)
	}
	if raw.Entities.Valid && raw.Entities.String != "" {
		_ = json.Unmarshal([]byte(raw.Entities.String), &state.Entities)
	}
	return state, nil
}

func (r *MCPStateRepository) Patch(ctx context.Context, roomID string, updates map[string]any, deletes []string) (*domain.RoomState, error) {
	state, err := r.Get(ctx, roomID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = &domain.RoomState{
			RoomID:      roomID,
			Summary:     "",
			ActiveTopic: "",
			State:       json.RawMessage("{}"),
		}
	}

	var stateMap map[string]any
	if len(state.State) > 0 && string(state.State) != "null" {
		_ = json.Unmarshal(state.State, &stateMap)
	}
	if stateMap == nil {
		stateMap = make(map[string]any)
	}

	for _, delKey := range deletes {
		delete(stateMap, delKey)
	}
	for k, v := range updates {
		stateMap[k] = v
	}

	newBytes, err := json.Marshal(stateMap)
	if err != nil {
		return nil, err
	}
	state.State = json.RawMessage(newBytes)

	if err := r.Upsert(ctx, state); err != nil {
		return nil, err
	}
	return state, nil
}

func (r *MCPStateRepository) Upsert(ctx context.Context, state *domain.RoomState) error {
	refBytes, _ := json.Marshal(state.References)
	if state.References == nil {
		refBytes = []byte("{}")
	}
	entBytes, _ := json.Marshal(state.Entities)
	if state.Entities == nil {
		entBytes = []byte("[]")
	}

	queryState := `
		INSERT INTO room_state (room_id, summary, active_topic, state, "references", "entities", last_summarized_seq, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (room_id) DO UPDATE SET
			summary = EXCLUDED.summary,
			active_topic = EXCLUDED.active_topic,
			state = EXCLUDED.state,
			"references" = EXCLUDED."references",
			"entities" = EXCLUDED."entities",
			last_summarized_seq = EXCLUDED.last_summarized_seq,
			updated_at = EXCLUDED.updated_at
	`
	if _, err := r.db.ExecContext(ctx, queryState, state.RoomID, state.Summary, state.ActiveTopic, state.State, string(refBytes), string(entBytes), state.LastSummarizedSeq); err != nil {
		return err
	}

	_, err := r.db.ExecContext(ctx, "UPDATE rooms SET updated_at = NOW() WHERE id = $1", state.RoomID)
	return err
}