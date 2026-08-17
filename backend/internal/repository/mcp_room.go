package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

// MCPRoomRepository executes project-scoped room queries for the MCP server.
type MCPRoomRepository struct {
	db *sqlx.DB
}

type mcpRawRoom struct {
	ID             string         `db:"id"`
	ProjectID      string         `db:"project_id"`
	ExternalUserID string         `db:"external_user_id"`
	Description    string         `db:"description"`
	ParentRoomID   sql.NullString `db:"parent_room_id"`
	CreatedAt      string         `db:"created_at"`
	UpdatedAt      string         `db:"updated_at"`
	StateSchema    sql.NullString `db:"state_schema"`
	DefaultPolicy  sql.NullString `db:"default_policy"`
	Status         string         `db:"status"`
	ArchiveS3Path  sql.NullString `db:"archive_s3_path"`
}

func NewMCPRoomRepository(db *sqlx.DB) *MCPRoomRepository {
	return &MCPRoomRepository{db: db}
}

func (r *MCPRoomRepository) Create(ctx context.Context, projectID string, externalUserID string, description string, parentRoomID *string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy, initialState json.RawMessage) (*domain.Room, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var room domain.Room
	queryRoom := `
		INSERT INTO rooms (project_id, external_user_id, description, parent_room_id, state_schema, default_policy, status)
		VALUES ($1, $2, $3, $4, $5, $6, 'active')
		RETURNING id, project_id, external_user_id, description, parent_room_id, created_at, updated_at, state_schema, default_policy, status, archive_s3_path
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
	var parentParam any = nil
	if parentRoomID != nil && *parentRoomID != "" {
		parentParam = *parentRoomID
	}

	var rr mcpRawRoom
	if err := tx.GetContext(ctx, &rr, queryRoom, projectID, externalUserID, description, parentParam, schemaParam, policyParam); err != nil {
		return nil, err
	}
	room = domain.Room{
		ID:             rr.ID,
		ProjectID:      rr.ProjectID,
		ExternalUserID: rr.ExternalUserID,
		Description:    rr.Description,
		CreatedAt:      rr.CreatedAt,
		UpdatedAt:      rr.UpdatedAt,
		Status:         rr.Status,
	}
	if rr.ParentRoomID.Valid {
		pID := rr.ParentRoomID.String
		room.ParentRoomID = &pID
	}
	if rr.ArchiveS3Path.Valid {
		pathStr := rr.ArchiveS3Path.String
		room.ArchiveS3Path = &pathStr
	}
	if rr.StateSchema.Valid {
		room.StateSchema = []byte(rr.StateSchema.String)
	}
	if rr.DefaultPolicy.Valid {
		var p domain.ContextPolicy
		_ = json.Unmarshal([]byte(rr.DefaultPolicy.String), &p)
		room.DefaultPolicy = &p
	}

	initialStateStr := "{}"
	if len(initialState) > 0 && string(initialState) != "null" && string(initialState) != "{}" {
		initialStateStr = string(initialState)
	}
	queryState := `
		INSERT INTO room_state (room_id, summary, active_topic, state)
		VALUES ($1, '', '', $2::jsonb)
	`
	if _, err := tx.ExecContext(ctx, queryState, room.ID, initialStateStr); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return &room, nil
}

func (r *MCPRoomRepository) GetByID(ctx context.Context, projectID string, roomID string) (*domain.Room, error) {
	query := `
		SELECT id, project_id, external_user_id, description, parent_room_id, created_at, updated_at, state_schema, default_policy, status, archive_s3_path
		FROM rooms
		WHERE project_id = $1 AND id = $2
	`
	var rr mcpRawRoom
	if err := r.db.GetContext(ctx, &rr, query, projectID, roomID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return mapRawRoom(&rr), nil
}

func (r *MCPRoomRepository) UpdateStatus(ctx context.Context, roomID string, status string) error {
	query := "UPDATE rooms SET status = $1, updated_at = NOW() WHERE id = $2"
	_, err := r.db.ExecContext(ctx, query, status, roomID)
	return err
}

func (r *MCPRoomRepository) Update(ctx context.Context, projectID string, roomID string, description *string, parentRoomID *string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy) (*domain.Room, error) {
	var currentRaw mcpRawRoom
	queryFetch := `
		SELECT id, project_id, external_user_id, description, parent_room_id, created_at, updated_at, state_schema, default_policy, status, archive_s3_path
		FROM rooms
		WHERE id = $1 AND project_id = $2
	`
	if err := r.db.GetContext(ctx, &currentRaw, queryFetch, roomID, projectID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	descParam := currentRaw.Description
	if description != nil {
		descParam = *description
	}

	var parentParam any = nil
	if currentRaw.ParentRoomID.Valid {
		parentParam = currentRaw.ParentRoomID.String
	}
	if parentRoomID != nil {
		if *parentRoomID == "" {
			parentParam = nil
		} else {
			parentParam = *parentRoomID
		}
	}

	schemaParam := currentRaw.StateSchema.String
	if len(stateSchema) > 0 && string(stateSchema) != "null" {
		schemaParam = string(stateSchema)
	}

	var policyParam any = nil
	if currentRaw.DefaultPolicy.Valid {
		policyParam = currentRaw.DefaultPolicy.String
	}
	if defaultPolicy != nil {
		pBytes, _ := json.Marshal(defaultPolicy)
		policyParam = string(pBytes)
	}

	queryUpdate := `
		UPDATE rooms
		SET description = $3, parent_room_id = $4, state_schema = $5, default_policy = $6, updated_at = NOW()
		WHERE id = $1 AND project_id = $2
		RETURNING id, project_id, external_user_id, description, parent_room_id, created_at, updated_at, state_schema, default_policy, status, archive_s3_path
	`
	var rr mcpRawRoom
	if err := r.db.GetContext(ctx, &rr, queryUpdate, roomID, projectID, descParam, parentParam, schemaParam, policyParam); err != nil {
		return nil, err
	}
	return mapRawRoom(&rr), nil
}

func (r *MCPRoomRepository) ListByProject(ctx context.Context, projectID string, externalUserID string, limit int, offset int) ([]domain.RoomOverview, int, error) {
	queryConds := "WHERE project_id = $1"
	args := []any{projectID}
	paramIdx := 2
	if externalUserID != "" {
		queryConds += fmt.Sprintf(" AND external_user_id = $%d", paramIdx)
		args = append(args, externalUserID)
		paramIdx++
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM rooms %s
	`, queryConds)
	var total int
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return nil, 0, err
	}

	selectQuery := fmt.Sprintf(`
		WITH room_metrics AS (
			SELECT
				r.id,
				COALESCE(r.external_user_id, '') AS external_user_id,
				COALESCE(r.description, '') AS description,
				r.parent_room_id,
				COALESCE(rs.active_topic, '') AS active_topic,
				COALESCE(mc.msg_count, 0) AS message_count,
				COALESCE(uc.total_tokens, 0) AS total_tokens_used,
				COALESCE(uc.total_cost, 0) AS total_cost,
				COALESCE(uc.avg_tokens, 0) AS avg_tokens_per_turn,
				r.updated_at AS last_active_at,
				r.project_id,
				r.status,
				r.archive_s3_path
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
		)
		SELECT id, external_user_id, description, parent_room_id, active_topic, message_count, total_tokens_used, total_cost, avg_tokens_per_turn, last_active_at, status, archive_s3_path
		FROM room_metrics
		%s
		ORDER BY last_active_at DESC
		LIMIT $%d OFFSET $%d
	`, queryConds, paramIdx, paramIdx+1)

	args = append(args, limit, offset)
	var overviews []domain.RoomOverview
	if err := r.db.SelectContext(ctx, &overviews, selectQuery, args...); err != nil {
		return nil, 0, err
	}
	if overviews == nil {
		overviews = []domain.RoomOverview{}
	}
	return overviews, total, nil
}

func (r *MCPRoomRepository) ListChildRooms(ctx context.Context, projectID string, parentRoomID string) ([]domain.ChildAgentOverview, error) {
	query := `
		SELECT
			r.id AS room_id,
			COALESCE(r.description, '') AS description,
			r.status,
			COALESCE(rs.active_topic, '') AS active_topic,
			COALESCE(rs.summary, '') AS summary
		FROM rooms r
		LEFT JOIN room_state rs ON rs.room_id = r.id
		WHERE r.project_id = $1 AND r.parent_room_id = $2
		ORDER BY r.created_at ASC
	`
	var children []domain.ChildAgentOverview
	if err := r.db.SelectContext(ctx, &children, query, projectID, parentRoomID); err != nil {
		return nil, err
	}
	if children == nil {
		children = []domain.ChildAgentOverview{}
	}
	return children, nil
}

func mapRawRoom(rr *mcpRawRoom) *domain.Room {
	room := &domain.Room{
		ID:             rr.ID,
		ProjectID:      rr.ProjectID,
		ExternalUserID: rr.ExternalUserID,
		Description:    rr.Description,
		CreatedAt:      rr.CreatedAt,
		UpdatedAt:      rr.UpdatedAt,
		Status:         rr.Status,
	}
	if rr.ParentRoomID.Valid {
		pID := rr.ParentRoomID.String
		room.ParentRoomID = &pID
	}
	if rr.ArchiveS3Path.Valid {
		pathStr := rr.ArchiveS3Path.String
		room.ArchiveS3Path = &pathStr
	}
	if rr.StateSchema.Valid && rr.StateSchema.String != "" {
		room.StateSchema = []byte(rr.StateSchema.String)
	}
	if rr.DefaultPolicy.Valid && rr.DefaultPolicy.String != "" {
		var p domain.ContextPolicy
		if json.Unmarshal([]byte(rr.DefaultPolicy.String), &p) == nil {
			room.DefaultPolicy = &p
		}
	}
	return room
}