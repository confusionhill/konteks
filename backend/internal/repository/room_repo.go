package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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
		INSERT INTO rooms (tenant_id, external_user_id, state_schema, default_policy, status)
		VALUES ($1, $2, $3, $4, 'active')
		RETURNING id, tenant_id, external_user_id, created_at, updated_at, state_schema, default_policy, status, archive_s3_path
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
		Status         string         `db:"status"`
		ArchiveS3Path  sql.NullString `db:"archive_s3_path"`
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
		Status:         rr.Status,
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
		SELECT id, tenant_id, external_user_id, created_at, updated_at, state_schema, default_policy, status, archive_s3_path
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
		Status         string         `db:"status"`
		ArchiveS3Path  sql.NullString `db:"archive_s3_path"`
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
		Status:         rr.Status,
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
		RoomID            string         `db:"room_id"`
		Summary           string         `db:"summary"`
		ActiveTopic       string         `db:"active_topic"`
		State             sql.NullString `db:"state"`
		References        sql.NullString `db:"references"`
		Entities          sql.NullString `db:"entities"`
		LastSummarizedSeq int64          `db:"last_summarized_seq"`
		UpdatedAt         string         `db:"updated_at"`
	}
	query := `
		SELECT room_id, summary, active_topic, state, "references", "entities", last_summarized_seq, updated_at
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
		INSERT INTO room_state (room_id, summary, active_topic, state, "references", "entities", last_summarized_seq, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (room_id) DO UPDATE SET
			summary = EXCLUDED.summary,
			active_topic = EXCLUDED.active_topic,
			state = EXCLUDED.state,
			"references" = EXCLUDED."references",
			"entities" = EXCLUDED."entities",
			last_summarized_seq = EXCLUDED.last_summarized_seq,
			updated_at = EXCLUDED.updated_at
	`
	now := time.Now()
	_, err = tx.ExecContext(ctx, queryState, state.RoomID, state.Summary, state.ActiveTopic, state.State, string(refBytes), string(entBytes), state.LastSummarizedSeq, now)
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

func (r *roomRepository) ListWithOverview(ctx context.Context, tenantID string, search string, status string, sortBy string, sortOrder string, minMessages int, limit int, offset int) ([]domain.RoomOverview, int, error) {
	// 1. Build ORDER BY safely to prevent SQL injection
	allowedSortFields := map[string]string{
		"id":               "id",
		"external_user_id": "external_user_id",
		"active_topic":     "active_topic",
		"message_count":    "message_count",
		"total_cost":       "total_cost",
		"last_active_at":   "last_active_at",
	}

	orderBy := "last_active_at"
	if field, ok := allowedSortFields[sortBy]; ok {
		orderBy = field
	}

	dir := "DESC"
	if strings.ToUpper(sortOrder) == "ASC" {
		dir = "ASC"
	}

	// 2. Build where conditions dynamically
	queryConds := ""
	args := []any{tenantID}
	paramIdx := 2

	if search != "" {
		queryConds += fmt.Sprintf(" AND (id::text ILIKE $%d OR external_user_id ILIKE $%d OR active_topic ILIKE $%d)", paramIdx, paramIdx, paramIdx)
		args = append(args, "%"+search+"%")
		paramIdx++
	}

	if status != "" {
		queryConds += fmt.Sprintf(" AND r.status = $%d", paramIdx)
		args = append(args, status)
		paramIdx++
	}

	if minMessages > 0 {
		queryConds += fmt.Sprintf(" AND message_count >= $%d", paramIdx)
		args = append(args, minMessages)
		paramIdx++
	}

	// 3. Count total rooms matching filters
	countQuery := fmt.Sprintf(`
		WITH room_metrics AS (
			SELECT
				r.id,
				COALESCE(r.external_user_id, '') AS external_user_id,
				COALESCE(rs.active_topic, '') AS active_topic,
				COALESCE(mc.msg_count, 0) AS message_count,
				r.tenant_id
			FROM rooms r
			LEFT JOIN room_state rs ON rs.room_id = r.id
			LEFT JOIN (
				SELECT room_id, COUNT(*) AS msg_count FROM messages GROUP BY room_id
			) mc ON mc.room_id = r.id
		)
		SELECT COUNT(*) FROM room_metrics WHERE tenant_id = $1 %s
	`, queryConds)

	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	// 4. Retrieve paginated list
	selectQuery := fmt.Sprintf(`
		WITH room_metrics AS (
			SELECT
				r.id,
				COALESCE(r.external_user_id, '') AS external_user_id,
				COALESCE(rs.active_topic, '') AS active_topic,
				COALESCE(mc.msg_count, 0) AS message_count,
				COALESCE(uc.total_tokens, 0) AS total_tokens_used,
				COALESCE(uc.total_cost, 0) AS total_cost,
				COALESCE(uc.avg_tokens, 0) AS avg_tokens_per_turn,
				r.updated_at AS last_active_at,
				r.tenant_id,
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
		SELECT id, external_user_id, active_topic, message_count, total_tokens_used, total_cost, avg_tokens_per_turn, last_active_at, status, archive_s3_path
		FROM room_metrics
		WHERE tenant_id = $1 %s
		ORDER BY %s %s
		LIMIT $%d OFFSET $%d
	`, queryConds, orderBy, dir, paramIdx, paramIdx+1)

	args = append(args, limit, offset)

	var overviews []domain.RoomOverview
	err = r.db.SelectContext(ctx, &overviews, selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	return overviews, total, nil
}

func (r *roomRepository) Update(ctx context.Context, tenantID string, roomID string, stateSchema json.RawMessage, defaultPolicy *domain.ContextPolicy) (*domain.Room, error) {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	type rawRoom struct {
		ID             string         `db:"id"`
		TenantID       string         `db:"tenant_id"`
		ExternalUserID string         `db:"external_user_id"`
		CreatedAt      string         `db:"created_at"`
		UpdatedAt      string         `db:"updated_at"`
		StateSchema    sql.NullString `db:"state_schema"`
		DefaultPolicy  sql.NullString `db:"default_policy"`
		Status         string         `db:"status"`
		ArchiveS3Path  sql.NullString `db:"archive_s3_path"`
	}

	// 1. Fetch current room values to build fallback values
	var currentRaw rawRoom
	queryFetch := `
		SELECT id, tenant_id, external_user_id, created_at, updated_at, state_schema, default_policy, status, archive_s3_path
		FROM rooms
		WHERE id = $1 AND tenant_id = $2
	`
	err = tx.GetContext(ctx, &currentRaw, queryFetch, roomID, tenantID)
	if err != nil {
		return nil, err
	}

	// 2. Map update parameters (use fallback if empty/nil)
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

	// 3. Perform update query
	queryUpdate := `
		UPDATE rooms
		SET state_schema = $3, default_policy = $4, updated_at = NOW()
		WHERE id = $1 AND tenant_id = $2
		RETURNING id, tenant_id, external_user_id, created_at, updated_at, state_schema, default_policy, status, archive_s3_path
	`

	var rr rawRoom
	err = tx.GetContext(ctx, &rr, queryUpdate, roomID, tenantID, schemaParam, policyParam)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	// Map back to domain.Room struct
	room := &domain.Room{
		ID:             rr.ID,
		TenantID:       rr.TenantID,
		ExternalUserID: rr.ExternalUserID,
		CreatedAt:      rr.CreatedAt,
		UpdatedAt:      rr.UpdatedAt,
		Status:         rr.Status,
	}

	if rr.ArchiveS3Path.Valid {
		pathStr := rr.ArchiveS3Path.String
		room.ArchiveS3Path = &pathStr
	}

	if rr.StateSchema.Valid && rr.StateSchema.String != "" {
		room.StateSchema = json.RawMessage(rr.StateSchema.String)
	}
	if rr.DefaultPolicy.Valid && rr.DefaultPolicy.String != "" {
		var policy domain.ContextPolicy
		if err := json.Unmarshal([]byte(rr.DefaultPolicy.String), &policy); err == nil {
			room.DefaultPolicy = &policy
		}
	}

	return room, nil
}

func (r *roomRepository) ListRoomsForArchival(ctx context.Context, inactiveDuration time.Duration, roomID string) ([]domain.Room, error) {
	var query string
	var args []interface{}
	if roomID != "" {
		query = `
			SELECT id, tenant_id, external_user_id, created_at, updated_at, state_schema, default_policy, status, archive_s3_path
			FROM rooms
			WHERE id = $1 AND status = 'active'
		`
		args = []interface{}{roomID}
	} else {
		query = `
			SELECT id, tenant_id, external_user_id, created_at, updated_at, state_schema, default_policy, status, archive_s3_path
			FROM rooms
			WHERE status = 'active' AND updated_at < $1
		`
		cutoff := time.Now().Add(-inactiveDuration)
		args = []interface{}{cutoff}
	}

	type rawRoom struct {
		ID             string         `db:"id"`
		TenantID       string         `db:"tenant_id"`
		ExternalUserID string         `db:"external_user_id"`
		CreatedAt      string         `db:"created_at"`
		UpdatedAt      string         `db:"updated_at"`
		StateSchema    sql.NullString `db:"state_schema"`
		DefaultPolicy  sql.NullString `db:"default_policy"`
		Status         string         `db:"status"`
		ArchiveS3Path  sql.NullString `db:"archive_s3_path"`
	}

	var rawList []rawRoom
	err := r.db.SelectContext(ctx, &rawList, query, args...)
	if err != nil {
		return nil, err
	}

	rooms := make([]domain.Room, len(rawList))
	for i, rr := range rawList {
		room := domain.Room{
			ID:             rr.ID,
			TenantID:       rr.TenantID,
			ExternalUserID: rr.ExternalUserID,
			CreatedAt:      rr.CreatedAt,
			UpdatedAt:      rr.UpdatedAt,
			Status:         rr.Status,
		}
		if rr.StateSchema.Valid && rr.StateSchema.String != "" {
			room.StateSchema = []byte(rr.StateSchema.String)
		}
		if rr.DefaultPolicy.Valid && rr.DefaultPolicy.String != "" {
			var p domain.ContextPolicy
			_ = json.Unmarshal([]byte(rr.DefaultPolicy.String), &p)
			room.DefaultPolicy = &p
		}
		if rr.ArchiveS3Path.Valid {
			pathStr := rr.ArchiveS3Path.String
			room.ArchiveS3Path = &pathStr
		}
		rooms[i] = room
	}

	return rooms, nil
}

func (r *roomRepository) UpdateStatus(ctx context.Context, roomID string, status string) error {
	query := "UPDATE rooms SET status = $1, updated_at = NOW() WHERE id = $2"
	_, err := r.db.ExecContext(ctx, query, status, roomID)
	return err
}

func (r *roomRepository) ArchiveRoomTx(ctx context.Context, roomID string, archiveS3Path string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Delete messages
	_, err = tx.ExecContext(ctx, "DELETE FROM messages WHERE room_id = $1", roomID)
	if err != nil {
		return err
	}

	// 2. Delete memories
	_, err = tx.ExecContext(ctx, "DELETE FROM memories WHERE room_id = $1", roomID)
	if err != nil {
		return err
	}

	// 3. Update room status to archived & set S3 path
	_, err = tx.ExecContext(ctx, "UPDATE rooms SET status = $1, archive_s3_path = $2, updated_at = NOW() WHERE id = $3", domain.RoomStatusArchived, archiveS3Path, roomID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (r *roomRepository) HydrateRoomTx(ctx context.Context, roomID string, state *domain.RoomState, messages []domain.Message, memories []domain.Memory) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Update room status to active and clear archive path
	_, err = tx.ExecContext(ctx, "UPDATE rooms SET status = $1, archive_s3_path = NULL, updated_at = NOW() WHERE id = $2", domain.RoomStatusActive, roomID)
	if err != nil {
		return err
	}

	// 2. Upsert RoomState
	if state != nil {
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
		_, err = tx.ExecContext(ctx, queryState, state.RoomID, state.Summary, state.ActiveTopic, state.State, string(refBytes), string(entBytes), state.LastSummarizedSeq)
		if err != nil {
			return err
		}
	}

	// 3. Insert Messages
	if len(messages) > 0 {
		queryMsg := `
			INSERT INTO messages (id, room_id, role, content, sequence, created_at, tokens_input, tokens_output, cost_total)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9
			WHERE NOT EXISTS (
				SELECT 1 FROM messages WHERE id = $1 OR (room_id = $2 AND sequence = $5)
			)
		`
		for _, msg := range messages {
			_, err = tx.ExecContext(ctx, queryMsg, msg.ID, msg.RoomID, msg.Role, msg.Content, msg.Sequence, msg.CreatedAt, msg.TokensInput, msg.TokensOutput, msg.CostTotal)
			if err != nil {
				return err
			}
		}
	}

	// 4. Insert Memories
	if len(memories) > 0 {
		queryMem := `
			INSERT INTO memories (id, tenant_id, room_id, external_user_id, content, embedding, metadata, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
			ON CONFLICT (id) DO NOTHING
		`
		for _, mem := range memories {
			vectorStr := float32SliceToVectorString(mem.Embedding)
			_, err = tx.ExecContext(ctx, queryMem, mem.ID, mem.TenantID, mem.RoomID, mem.ExternalUserID, mem.Content, vectorStr, mem.Metadata)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func float32SliceToVectorString(slice []float32) string {
	var builder strings.Builder
	builder.WriteByte('[')
	for i, v := range slice {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(fmt.Sprintf("%g", v))
	}
	builder.WriteByte(']')
	return builder.String()
}

