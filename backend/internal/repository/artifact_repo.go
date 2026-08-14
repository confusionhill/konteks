package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type artifactRepository struct {
	db *sqlx.DB
}

func NewArtifactRepository(db *sqlx.DB) domain.ArtifactRepository {
	return &artifactRepository{db: db}
}

func (r *artifactRepository) CreateGroup(ctx context.Context, group *domain.ArtifactGroup) error {
	query := `
		INSERT INTO artifact_groups (project_id, name, folder_path, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`
	type Inserted struct {
		ID        string    `db:"id"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	var ins Inserted
	err := r.db.GetContext(ctx, &ins, query, group.ProjectID, group.Name, group.FolderPath, group.Description)
	if err != nil {
		return fmt.Errorf("failed to create artifact group: %w", err)
	}

	group.ID = ins.ID
	group.CreatedAt = ins.CreatedAt
	group.UpdatedAt = ins.UpdatedAt
	return nil
}

func (r *artifactRepository) GetGroupByName(ctx context.Context, projectID string, name string) (*domain.ArtifactGroup, error) {
	query := `
		SELECT id, project_id, name, folder_path, description, created_at, updated_at
		FROM artifact_groups
		WHERE project_id = $1 AND name = $2
	`
	var g domain.ArtifactGroup
	err := r.db.GetContext(ctx, &g, query, projectID, name)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *artifactRepository) ListGroups(ctx context.Context, projectID string) ([]domain.ArtifactGroup, error) {
	query := `
		SELECT id, project_id, name, folder_path, description, created_at, updated_at
		FROM artifact_groups
		WHERE project_id = $1
		ORDER BY name ASC
	`
	var groups []domain.ArtifactGroup
	err := r.db.SelectContext(ctx, &groups, query, projectID)
	if err != nil {
		return nil, err
	}
	return groups, nil
}

func (r *artifactRepository) CreateArtifact(ctx context.Context, artifact *domain.Artifact) error {
	query := `
		INSERT INTO artifacts (project_id, group_id, file_path, title, s3_key, content_hash, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	metaJSON := artifact.Metadata
	if len(metaJSON) == 0 {
		metaJSON = json.RawMessage("{}")
	}

	type Inserted struct {
		ID        string    `db:"id"`
		CreatedAt time.Time `db:"created_at"`
		UpdatedAt time.Time `db:"updated_at"`
	}
	var ins Inserted
	err := r.db.GetContext(ctx, &ins, query,
		artifact.ProjectID,
		artifact.GroupID,
		artifact.FilePath,
		artifact.Title,
		artifact.S3Key,
		artifact.ContentHash,
		metaJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to create artifact: %w", err)
	}

	artifact.ID = ins.ID
	artifact.CreatedAt = ins.CreatedAt
	artifact.UpdatedAt = ins.UpdatedAt
	return nil
}

func (r *artifactRepository) UpdateArtifact(ctx context.Context, artifact *domain.Artifact) error {
	query := `
		UPDATE artifacts
		SET group_id = $1, title = $2, s3_key = $3, content_hash = $4, metadata = $5, updated_at = NOW()
		WHERE id = $6 AND project_id = $7
		RETURNING updated_at
	`
	metaJSON := artifact.Metadata
	if len(metaJSON) == 0 {
		metaJSON = json.RawMessage("{}")
	}

	var updatedAt time.Time
	err := r.db.GetContext(ctx, &updatedAt, query,
		artifact.GroupID,
		artifact.Title,
		artifact.S3Key,
		artifact.ContentHash,
		metaJSON,
		artifact.ID,
		artifact.ProjectID,
	)
	if err != nil {
		return fmt.Errorf("failed to update artifact: %w", err)
	}
	artifact.UpdatedAt = updatedAt
	return nil
}

type dbArtifactRow struct {
	ID          string         `db:"id"`
	ProjectID   string         `db:"project_id"`
	GroupID     sql.NullString `db:"group_id"`
	GroupName   sql.NullString `db:"group_name"`
	FilePath    string         `db:"file_path"`
	Title       string         `db:"title"`
	S3Key       string         `db:"s3_key"`
	ContentHash string         `db:"content_hash"`
	Metadata    []byte         `db:"metadata"`
	ChunksCount int            `db:"chunks_count"`
	CreatedAt   time.Time      `db:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at"`
}

func mapDBArtifactRow(row dbArtifactRow) domain.Artifact {
	var groupID *string
	if row.GroupID.Valid {
		g := row.GroupID.String
		groupID = &g
	}
	groupName := ""
	if row.GroupName.Valid {
		groupName = row.GroupName.String
	}

	metaJSON := json.RawMessage("{}")
	if len(row.Metadata) > 0 {
		metaJSON = json.RawMessage(row.Metadata)
	}

	return domain.Artifact{
		ID:          row.ID,
		ProjectID:   row.ProjectID,
		GroupID:     groupID,
		GroupName:   groupName,
		FilePath:    row.FilePath,
		Title:       row.Title,
		S3Key:       row.S3Key,
		ContentHash: row.ContentHash,
		Metadata:    metaJSON,
		ChunksCount: row.ChunksCount,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func (r *artifactRepository) GetArtifactByID(ctx context.Context, artifactID string) (*domain.Artifact, error) {
	query := `
		SELECT a.id, a.project_id, a.group_id, g.name as group_name, a.file_path, a.title, a.s3_key, a.content_hash, a.metadata, a.created_at, a.updated_at,
		       (SELECT COUNT(*) FROM artifact_chunks c WHERE c.artifact_id = a.id) as chunks_count
		FROM artifacts a
		LEFT JOIN artifact_groups g ON a.group_id = g.id
		WHERE a.id = $1
	`
	var row dbArtifactRow
	err := r.db.GetContext(ctx, &row, query, artifactID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	art := mapDBArtifactRow(row)
	return &art, nil
}

func (r *artifactRepository) GetArtifactByPath(ctx context.Context, projectID string, filePath string) (*domain.Artifact, error) {
	query := `
		SELECT a.id, a.project_id, a.group_id, g.name as group_name, a.file_path, a.title, a.s3_key, a.content_hash, a.metadata, a.created_at, a.updated_at,
		       (SELECT COUNT(*) FROM artifact_chunks c WHERE c.artifact_id = a.id) as chunks_count
		FROM artifacts a
		LEFT JOIN artifact_groups g ON a.group_id = g.id
		WHERE a.project_id = $1 AND a.file_path = $2
	`
	var row dbArtifactRow
	err := r.db.GetContext(ctx, &row, query, projectID, filePath)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	art := mapDBArtifactRow(row)
	return &art, nil
}

func (r *artifactRepository) ListArtifacts(ctx context.Context, projectID string, groupName string, limit int, offset int) ([]domain.Artifact, int, error) {
	var countQuery string
	var listQuery string
	var args []any

	if groupName != "" {
		countQuery = `
			SELECT COUNT(*) 
			FROM artifacts a
			LEFT JOIN artifact_groups g ON a.group_id = g.id
			WHERE a.project_id = $1 AND g.name = $2
		`
		listQuery = `
			SELECT a.id, a.project_id, a.group_id, g.name as group_name, a.file_path, a.title, a.s3_key, a.content_hash, a.metadata, a.created_at, a.updated_at,
			       (SELECT COUNT(*) FROM artifact_chunks c WHERE c.artifact_id = a.id) as chunks_count
			FROM artifacts a
			LEFT JOIN artifact_groups g ON a.group_id = g.id
			WHERE a.project_id = $1 AND g.name = $2
			ORDER BY a.file_path ASC
			LIMIT $3 OFFSET $4
		`
		args = []any{projectID, groupName}
	} else {
		countQuery = `SELECT COUNT(*) FROM artifacts WHERE project_id = $1`
		listQuery = `
			SELECT a.id, a.project_id, a.group_id, g.name as group_name, a.file_path, a.title, a.s3_key, a.content_hash, a.metadata, a.created_at, a.updated_at,
			       (SELECT COUNT(*) FROM artifact_chunks c WHERE c.artifact_id = a.id) as chunks_count
			FROM artifacts a
			LEFT JOIN artifact_groups g ON a.group_id = g.id
			WHERE a.project_id = $1
			ORDER BY a.file_path ASC
			LIMIT $2 OFFSET $3
		`
		args = []any{projectID}
	}

	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, err
	}

	var rows []dbArtifactRow
	if groupName != "" {
		err = r.db.SelectContext(ctx, &rows, listQuery, projectID, groupName, limit, offset)
	} else {
		err = r.db.SelectContext(ctx, &rows, listQuery, projectID, limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}

	artifacts := make([]domain.Artifact, len(rows))
	for i, r := range rows {
		artifacts[i] = mapDBArtifactRow(r)
	}
	return artifacts, total, nil
}

func (r *artifactRepository) DeleteArtifact(ctx context.Context, artifactID string) error {
	query := `DELETE FROM artifacts WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, artifactID)
	return err
}

func (r *artifactRepository) CreateChunks(ctx context.Context, chunks []domain.ArtifactChunk) error {
	if len(chunks) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	query := `
		INSERT INTO artifact_chunks (artifact_id, project_id, chunk_index, section_heading, content, embedding)
		VALUES ($1, $2, $3, $4, $5, $6::vector)
	`
	stmt, err := tx.PreparexContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, chunk := range chunks {
		embStr := float32ArrayToString(chunk.Embedding)
		_, err := stmt.ExecContext(ctx, chunk.ArtifactID, chunk.ProjectID, chunk.ChunkIndex, chunk.SectionHeading, chunk.Content, embStr)
		if err != nil {
			return fmt.Errorf("failed inserting chunk index %d: %w", chunk.ChunkIndex, err)
		}
	}

	return tx.Commit()
}

func (r *artifactRepository) DeleteChunksByArtifact(ctx context.Context, artifactID string) error {
	query := `DELETE FROM artifact_chunks WHERE artifact_id = $1`
	_, err := r.db.ExecContext(ctx, query, artifactID)
	return err
}

func (r *artifactRepository) SearchChunksVector(
	ctx context.Context,
	projectID string,
	queryEmbedding []float32,
	limit int,
	excludedArtifactIDs []string,
	excludedGroupNames []string,
) ([]domain.ArtifactSearchResult, error) {
	embStr := float32ArrayToString(queryEmbedding)

	query := `
		SELECT c.id, c.artifact_id, c.project_id, c.chunk_index, c.section_heading, c.content,
		       a.file_path, a.title, a.s3_key, a.content_hash, a.metadata, g.name as group_name,
		       (1 - (c.embedding <=> $2::vector)) as relevance_score
		FROM artifact_chunks c
		JOIN artifacts a ON c.artifact_id = a.id
		LEFT JOIN artifact_groups g ON a.group_id = g.id
		WHERE c.project_id = $1
		  AND ($3::uuid[] IS NULL OR c.artifact_id != ALL($3::uuid[]))
		  AND ($4::text[] IS NULL OR g.name IS NULL OR g.name != ALL($4::text[]))
		ORDER BY c.embedding <=> $2::vector ASC
		LIMIT $5
	`

	var artifactIDsParam pq.StringArray = nil
	if len(excludedArtifactIDs) > 0 {
		artifactIDsParam = pq.StringArray(excludedArtifactIDs)
	}
	var groupNamesParam pq.StringArray = nil
	if len(excludedGroupNames) > 0 {
		groupNamesParam = pq.StringArray(excludedGroupNames)
	}

	type dbSearchResultRow struct {
		ID             string         `db:"id"`
		ArtifactID     string         `db:"artifact_id"`
		ProjectID      string         `db:"project_id"`
		ChunkIndex     int            `db:"chunk_index"`
		SectionHeading string         `db:"section_heading"`
		Content        string         `db:"content"`
		FilePath       string         `db:"file_path"`
		Title          string         `db:"title"`
		S3Key          string         `db:"s3_key"`
		ContentHash    string         `db:"content_hash"`
		Metadata       []byte         `db:"metadata"`
		GroupName      sql.NullString `db:"group_name"`
		RelevanceScore float64        `db:"relevance_score"`
	}

	var rows []dbSearchResultRow
	err := r.db.SelectContext(ctx, &rows, query, projectID, embStr, artifactIDsParam, groupNamesParam, limit)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	results := make([]domain.ArtifactSearchResult, len(rows))
	for i, row := range rows {
		groupName := ""
		if row.GroupName.Valid {
			groupName = row.GroupName.String
		}
		metaJSON := json.RawMessage("{}")
		if len(row.Metadata) > 0 {
			metaJSON = json.RawMessage(row.Metadata)
		}

		results[i] = domain.ArtifactSearchResult{
			Chunk: domain.ArtifactChunk{
				ID:             row.ID,
				ArtifactID:     row.ArtifactID,
				ProjectID:      row.ProjectID,
				ChunkIndex:     row.ChunkIndex,
				SectionHeading: row.SectionHeading,
				Content:        row.Content,
			},
			Artifact: domain.Artifact{
				ID:          row.ArtifactID,
				ProjectID:   row.ProjectID,
				GroupName:   groupName,
				FilePath:    row.FilePath,
				Title:       row.Title,
				S3Key:       row.S3Key,
				ContentHash: row.ContentHash,
				Metadata:    metaJSON,
			},
			RelevanceScore: row.RelevanceScore,
		}
	}

	return results, nil
}

func (r *artifactRepository) AddReferences(ctx context.Context, sourceArtifactID string, targetArtifactIDs []string, relationshipType string) error {
	if len(targetArtifactIDs) == 0 {
		return nil
	}
	if relationshipType == "" {
		relationshipType = "refers_to"
	}

	query := `
		INSERT INTO artifact_references (source_artifact_id, target_artifact_id, relationship_type)
		VALUES ($1, $2, $3)
		ON CONFLICT (source_artifact_id, target_artifact_id) DO UPDATE SET relationship_type = EXCLUDED.relationship_type
	`
	for _, targetID := range targetArtifactIDs {
		if targetID == sourceArtifactID {
			continue // avoid self-loops
		}
		_, err := r.db.ExecContext(ctx, query, sourceArtifactID, targetID, relationshipType)
		if err != nil {
			return fmt.Errorf("failed adding reference %s -> %s: %w", sourceArtifactID, targetID, err)
		}
	}
	return nil
}

func (r *artifactRepository) GetOutgoingReferences(ctx context.Context, sourceArtifactID string) ([]domain.ArtifactReference, error) {
	query := `
		SELECT source_artifact_id, target_artifact_id, relationship_type, created_at
		FROM artifact_references
		WHERE source_artifact_id = $1
	`
	var refs []domain.ArtifactReference
	err := r.db.SelectContext(ctx, &refs, query, sourceArtifactID)
	if err != nil {
		return nil, err
	}
	return refs, nil
}

func (r *artifactRepository) GetKnowledgeGraph(ctx context.Context, projectID string) (*domain.KnowledgeGraph, error) {
	nodesQuery := `
		SELECT a.id, a.file_path, a.title, COALESCE(g.name, '') as group_name
		FROM artifacts a
		LEFT JOIN artifact_groups g ON a.group_id = g.id
		WHERE a.project_id = $1
		ORDER BY a.file_path ASC
	`
	type dbNode struct {
		ID        string `db:"id"`
		FilePath  string `db:"file_path"`
		Title     string `db:"title"`
		GroupName string `db:"group_name"`
	}
	var dbNodes []dbNode
	err := r.db.SelectContext(ctx, &dbNodes, nodesQuery, projectID)
	if err != nil {
		return nil, err
	}

	edgesQuery := `
		SELECT r.source_artifact_id as source, r.target_artifact_id as target, r.relationship_type as relationship
		FROM artifact_references r
		JOIN artifacts a ON r.source_artifact_id = a.id
		WHERE a.project_id = $1
	`
	type dbEdge struct {
		Source       string `db:"source"`
		Target       string `db:"target"`
		Relationship string `db:"relationship"`
	}
	var dbEdges []dbEdge
	err = r.db.SelectContext(ctx, &dbEdges, edgesQuery, projectID)
	if err != nil {
		return nil, err
	}

	nodes := make([]domain.GraphNode, len(dbNodes))
	for i, n := range dbNodes {
		nodes[i] = domain.GraphNode{
			ID:        n.ID,
			FilePath:  n.FilePath,
			Title:     n.Title,
			GroupName: n.GroupName,
		}
	}
	edges := make([]domain.GraphEdge, len(dbEdges))
	for i, e := range dbEdges {
		edges[i] = domain.GraphEdge{
			Source:       e.Source,
			Target:       e.Target,
			Relationship: e.Relationship,
		}
	}

	return &domain.KnowledgeGraph{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func (r *artifactRepository) DeleteReferencesBySource(ctx context.Context, sourceArtifactID string) error {
	query := `DELETE FROM artifact_references WHERE source_artifact_id = $1`
	_, err := r.db.ExecContext(ctx, query, sourceArtifactID)
	return err
}

func (r *artifactRepository) AddRoomBlacklist(ctx context.Context, blacklist *domain.RoomArtifactBlacklist) error {
	query := `
		INSERT INTO room_artifact_blacklists (room_id, artifact_id, group_name)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`
	type Inserted struct {
		ID        string    `db:"id"`
		CreatedAt time.Time `db:"created_at"`
	}
	var ins Inserted
	err := r.db.GetContext(ctx, &ins, query, blacklist.RoomID, blacklist.ArtifactID, blacklist.GroupName)
	if err != nil {
		return fmt.Errorf("failed adding room blacklist: %w", err)
	}

	blacklist.ID = ins.ID
	blacklist.CreatedAt = ins.CreatedAt
	return nil
}

func (r *artifactRepository) RemoveRoomBlacklist(ctx context.Context, blacklistID string) error {
	query := `DELETE FROM room_artifact_blacklists WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, blacklistID)
	return err
}

func (r *artifactRepository) GetRoomBlacklists(ctx context.Context, roomID string) ([]domain.RoomArtifactBlacklist, error) {
	query := `
		SELECT id, room_id, artifact_id, group_name, created_at
		FROM room_artifact_blacklists
		WHERE room_id = $1
	`
	type dbBlacklist struct {
		ID        string         `db:"id"`
		RoomID    string         `db:"room_id"`
		ArtifactID sql.NullString `db:"artifact_id"`
		GroupName sql.NullString `db:"group_name"`
		CreatedAt time.Time      `db:"created_at"`
	}
	var rows []dbBlacklist
	err := r.db.SelectContext(ctx, &rows, query, roomID)
	if err != nil {
		return nil, err
	}

	blacklists := make([]domain.RoomArtifactBlacklist, len(rows))
	for i, row := range rows {
		var artID *string
		if row.ArtifactID.Valid {
			id := row.ArtifactID.String
			artID = &id
		}
		var grpName *string
		if row.GroupName.Valid {
			name := row.GroupName.String
			grpName = &name
		}
		blacklists[i] = domain.RoomArtifactBlacklist{
			ID:        row.ID,
			RoomID:    row.RoomID,
			ArtifactID: artID,
			GroupName: grpName,
			CreatedAt: row.CreatedAt,
		}
	}
	return blacklists, nil
}
