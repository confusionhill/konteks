package domain

import "context"

// VectorStore abstracts vector insert/search/delete operations.
// Cloudflare Vectorize is the single source of truth for embeddings.
type VectorStore interface {
	// UpsertMemoryVector upserts a single memory's vector embedding with metadata.
	// The vector ID is memory.ID. Metadata includes room_id, external_user_id, project_id, status.
	UpsertMemoryVector(ctx context.Context, memory *Memory) error

	// UpdateMemoryMetadata updates the metadata of an existing memory vector.
	// Used to keep status in sync (superseded / restored) with Postgres.
	UpdateMemoryMetadata(ctx context.Context, memoryID string, metadata map[string]any) error

	// SearchMemoryVectors performs cosine similarity search for memories.
	// Returns matched vector IDs and cosine similarity scores (0-1, higher = more similar).
	// Filters by projectID, room_id, and optionally external_user_id. Only returns active memories.
	SearchMemoryVectors(ctx context.Context, projectID string, roomID string, externalUserID *string, queryEmbedding []float32, limit int) ([]VectorMatch, error)

	// DeleteMemoryVectors removes memory vectors by their IDs.
	DeleteMemoryVectors(ctx context.Context, ids []string) error

	// UpsertArtifactChunkVectors upserts artifact chunk vectors in batch.
	// Each chunk's vector ID is chunk.ID. Metadata includes project_id, artifact_id, group_name.
	UpsertArtifactChunkVectors(ctx context.Context, chunks []ArtifactChunk, groupName string) error

	// SearchArtifactChunkVectors performs cosine similarity search for artifact chunks.
	// Filters by project_id, excludes specified artifact IDs and group names.
	SearchArtifactChunkVectors(ctx context.Context, projectID string, queryEmbedding []float32, limit int, excludedArtifactIDs []string, excludedGroupNames []string) ([]VectorMatch, error)

	// DeleteArtifactChunkVectors removes artifact chunk vectors by their IDs.
	DeleteArtifactChunkVectors(ctx context.Context, ids []string) error
}

// VectorMatch represents a single result from a vector similarity search.
type VectorMatch struct {
	ID    string  // The vector ID (maps to memory.id or artifact_chunk.id)
	Score float64 // Cosine similarity score (0-1, higher = more similar)
}
