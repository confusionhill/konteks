package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/jmoiron/sqlx"
)

type artifactRepository struct {
	db          *sqlx.DB
	vectorStore domain.VectorStore
}

func NewArtifactRepository(db *sqlx.DB, vectorStore domain.VectorStore) domain.ArtifactRepository {
	return &artifactRepository{db: db, vectorStore: vectorStore}
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
		INSERT INTO artifacts (id, project_id, group_id, file_path, title, s3_key, content_hash, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
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
		artifact.ID,
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
		WHERE a.project_id = $1 AND (a.file_path = $2 OR a.file_path LIKE '%/' || $2)
		ORDER BY (CASE WHEN a.file_path = $2 THEN 0 ELSE 1 END)
		LIMIT 1
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

	artifacts := make([]domain.Artifact, 0, len(rows))
	for _, r := range rows {
		artifacts = append(artifacts, mapDBArtifactRow(r))
	}
	return artifacts, total, nil
}

func (r *artifactRepository) DeleteArtifact(ctx context.Context, artifactID string) error {
	if r.vectorStore != nil {
		chunkQuery := `SELECT id FROM artifact_chunks WHERE artifact_id = $1`
		var ids []string
		if err := r.db.SelectContext(ctx, &ids, chunkQuery, artifactID); err != nil {
			return err
		}
		if len(ids) > 0 {
			if errVec := r.vectorStore.DeleteArtifactChunkVectors(ctx, ids); errVec != nil {
				log.Printf("[ArtifactRepo Warning]: failed to purge artifact chunk vectors: %v", errVec)
			}
		}
	}

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
		INSERT INTO artifact_chunks (id, artifact_id, project_id, chunk_index, section_heading, content, vector_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	stmt, err := tx.PreparexContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, chunk := range chunks {
		status := chunk.VectorStatus
		if status == "" {
			status = domain.ChunkVectorStatusIndexed
			if len(chunk.Embedding) == 0 {
				status = domain.ChunkVectorStatusPending
			}
		}
		_, err := stmt.ExecContext(ctx, chunk.ID, chunk.ArtifactID, chunk.ProjectID, chunk.ChunkIndex, chunk.SectionHeading, chunk.Content, status)
		if err != nil {
			return fmt.Errorf("failed inserting chunk index %d: %w", chunk.ChunkIndex, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	if r.vectorStore != nil {
		if errVec := r.vectorStore.UpsertArtifactChunkVectors(ctx, chunks, ""); errVec != nil {
			log.Printf("[ArtifactRepo Warning]: failed to upsert artifact chunks to vector store: %v", errVec)
		}
	}

	return nil
}

// ListPendingChunks returns chunks whose embedding still needs to be generated,
// oldest first, capped at limit.
func (r *artifactRepository) ListPendingChunks(ctx context.Context, limit int) ([]domain.ArtifactChunk, error) {
	query := `
		SELECT id, artifact_id, project_id, chunk_index, section_heading, content, vector_status, created_at
		FROM artifact_chunks
		WHERE vector_status = $1
		ORDER BY created_at ASC
		LIMIT $2
	`
	var rows []struct {
		ID             string    `db:"id"`
		ArtifactID     string    `db:"artifact_id"`
		ProjectID      string    `db:"project_id"`
		ChunkIndex     int       `db:"chunk_index"`
		SectionHeading string    `db:"section_heading"`
		Content        string    `db:"content"`
		VectorStatus   string    `db:"vector_status"`
		CreatedAt      time.Time `db:"created_at"`
	}
	if err := r.db.SelectContext(ctx, &rows, query, domain.ChunkVectorStatusPending, limit); err != nil {
		return nil, err
	}

	chunks := make([]domain.ArtifactChunk, 0, len(rows))
	for _, row := range rows {
		chunks = append(chunks, domain.ArtifactChunk{
			ID:             row.ID,
			ArtifactID:     row.ArtifactID,
			ProjectID:      row.ProjectID,
			ChunkIndex:     row.ChunkIndex,
			SectionHeading: row.SectionHeading,
			Content:        row.Content,
			VectorStatus:   row.VectorStatus,
			CreatedAt:      row.CreatedAt,
		})
	}
	return chunks, nil
}

// UpsertChunkVectors pushes embeddings for existing chunks to the vector store
// without touching their Postgres rows.
func (r *artifactRepository) UpsertChunkVectors(ctx context.Context, chunks []domain.ArtifactChunk, groupName string) error {
	if r.vectorStore == nil {
		return nil
	}
	return r.vectorStore.UpsertArtifactChunkVectors(ctx, chunks, groupName)
}

// MarkChunksIndexed flips pending chunks to indexed after their embeddings
// have been successfully pushed to the vector store.
func (r *artifactRepository) MarkChunksIndexed(ctx context.Context, chunkIDs []string) error {
	if len(chunkIDs) == 0 {
		return nil
	}
	query := `UPDATE artifact_chunks SET vector_status = $1 WHERE id = ANY($2::uuid[])`
	_, err := r.db.ExecContext(ctx, query, domain.ChunkVectorStatusIndexed, chunkIDs)
	return err
}

func (r *artifactRepository) DeleteChunksByArtifact(ctx context.Context, artifactID string) error {
	query := `
		SELECT id
		FROM artifact_chunks
		WHERE artifact_id = $1
	`
	var ids []string
	if err := r.db.SelectContext(ctx, &ids, query, artifactID); err != nil {
		return err
	}

	if r.vectorStore != nil && len(ids) > 0 {
		if errVec := r.vectorStore.DeleteArtifactChunkVectors(ctx, ids); errVec != nil {
			log.Printf("[ArtifactRepo Warning]: failed to purge artifact chunk vectors: %v", errVec)
		}
	}

	deleteQuery := `DELETE FROM artifact_chunks WHERE artifact_id = $1`
	_, err := r.db.ExecContext(ctx, deleteQuery, artifactID)
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
	if len(queryEmbedding) == 0 {
		return []domain.ArtifactSearchResult{}, nil
	}
	if r.vectorStore == nil {
		return nil, fmt.Errorf("vector store not configured")
	}

	matches, err := r.vectorStore.SearchArtifactChunkVectors(ctx, projectID, queryEmbedding, limit, excludedArtifactIDs, excludedGroupNames)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}
	if len(matches) == 0 {
		return []domain.ArtifactSearchResult{}, nil
	}

	ids := make([]string, 0, len(matches))
	scoreByID := make(map[string]float64, len(matches))
	for _, m := range matches {
		ids = append(ids, m.ID)
		scoreByID[m.ID] = m.Score
	}

	chunks, err := r.getChunksByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}

	results := make([]domain.ArtifactSearchResult, 0, len(matches))
	for _, id := range ids {
		ch, ok := chunks[id]
		if !ok {
			continue
		}
		results = append(results, domain.ArtifactSearchResult{
			Chunk:          ch.Chunk,
			Artifact:       ch.Artifact,
			RelevanceScore: scoreByID[id],
		})
	}
	return results, nil
}

type artifactChunkRow struct {
	Chunk    domain.ArtifactChunk
	Artifact domain.Artifact
}

func (r *artifactRepository) getChunksByIDs(ctx context.Context, ids []string) (map[string]artifactChunkRow, error) {
	query := `
		SELECT c.id, c.artifact_id, c.project_id, c.chunk_index, c.section_heading, c.content,
		       a.file_path, a.title, a.s3_key, a.content_hash, a.metadata, COALESCE(g.name, '') as group_name
		FROM artifact_chunks c
		JOIN artifacts a ON c.artifact_id = a.id
		LEFT JOIN artifact_groups g ON a.group_id = g.id
		WHERE c.id = ANY($1::uuid[])
	`

	type dbRow struct {
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
		GroupName      string         `db:"group_name"`
	}

	var rows []dbRow
	if err := r.db.SelectContext(ctx, &rows, query, ids); err != nil {
		return nil, err
	}

	result := make(map[string]artifactChunkRow, len(rows))
	for _, row := range rows {
		metaJSON := json.RawMessage("{}")
		if len(row.Metadata) > 0 {
			metaJSON = json.RawMessage(row.Metadata)
		}
		result[row.ID] = artifactChunkRow{
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
				GroupName:   row.GroupName,
				FilePath:    row.FilePath,
				Title:       row.Title,
				S3Key:       row.S3Key,
				ContentHash: row.ContentHash,
				Metadata:    metaJSON,
			},
		}
	}
	return result, nil
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

	nodes := make([]domain.GraphNode, 0, len(dbNodes))
	for _, n := range dbNodes {
		nodes = append(nodes, domain.GraphNode{
			ID:        n.ID,
			FilePath:  n.FilePath,
			Title:     n.Title,
			GroupName: n.GroupName,
		})
	}
	edges := make([]domain.GraphEdge, 0, len(dbEdges))
	for _, e := range dbEdges {
		edges = append(edges, domain.GraphEdge{
			Source:       e.Source,
			Target:       e.Target,
			Relationship: e.Relationship,
		})
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
