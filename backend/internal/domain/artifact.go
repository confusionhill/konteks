package domain

import (
	"context"
	"encoding/json"
	"time"
)

type ArtifactGroup struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Name        string    `json:"name"`
	FolderPath  string    `json:"folder_path"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Artifact struct {
	ID          string          `json:"id"`
	ProjectID   string          `json:"project_id"`
	GroupID     *string         `json:"group_id,omitempty"`
	GroupName   string          `json:"group_name,omitempty"`
	FilePath    string          `json:"file_path"`
	Title       string          `json:"title"`
	S3Key       string          `json:"s3_key"`
	ContentHash string          `json:"content_hash"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
	Content     string          `json:"content,omitempty"`
	ChunksCount int             `json:"chunks_count,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ChunkVectorStatusPending means the chunk's embedding failed to generate and
// has been scheduled for a background retry.
const ChunkVectorStatusPending = "pending"

// ChunkVectorStatusIndexed means the chunk's embedding has been successfully
// pushed to the vector store.
const ChunkVectorStatusIndexed = "indexed"

type ArtifactChunk struct {
	ID             string    `json:"id"`
	ArtifactID     string    `json:"artifact_id"`
	ProjectID      string    `json:"project_id"`
	ChunkIndex     int       `json:"chunk_index"`
	SectionHeading string    `json:"section_heading"`
	Content        string    `json:"content"`
	Embedding      []float32 `json:"embedding,omitempty"`
	VectorStatus   string    `json:"vector_status,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type ArtifactReference struct {
	SourceArtifactID string    `json:"source_artifact_id"`
	TargetArtifactID string    `json:"target_artifact_id"`
	RelationshipType string    `json:"relationship_type"`
	CreatedAt        time.Time `json:"created_at"`
}

type RoomArtifactBlacklist struct {
	ID        string    `json:"id"`
	RoomID    string    `json:"room_id"`
	ArtifactID *string   `json:"artifact_id,omitempty"`
	GroupName *string   `json:"group_name,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ArtifactContext struct {
	ArtifactID      string             `json:"artifact_id"`
	FilePath        string             `json:"file_path"`
	Title           string             `json:"title"`
	SectionHeading  string             `json:"section_heading,omitempty"`
	Content         string             `json:"content"`
	RelevanceScore  float64            `json:"relevance_score"`
	GraphReferences []GraphReference   `json:"graph_references,omitempty"`
}

type GraphReference struct {
	ArtifactID       string `json:"artifact_id"`
	FilePath         string `json:"file_path"`
	Title            string `json:"title"`
	RelationshipType string `json:"relationship_type"`
	Content          string `json:"content,omitempty"`
}

type GraphNode struct {
	ID        string `json:"id"`
	FilePath  string `json:"file_path"`
	Title     string `json:"title"`
	GroupName string `json:"group_name,omitempty"`
}

type GraphEdge struct {
	Source       string `json:"source"`
	Target       string `json:"target"`
	Relationship string `json:"relationship"`
}

type KnowledgeGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type CreateArtifactRequest struct {
	FilePath   string          `json:"file_path"`
	Title      string          `json:"title"`
	Content    string          `json:"content"`
	GroupName  string          `json:"group_name,omitempty"`
	References []string        `json:"references,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

type UpdateArtifactRequest struct {
	Title      string          `json:"title"`
	Content    string          `json:"content"`
	GroupName  string          `json:"group_name,omitempty"`
	References []string        `json:"references,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
}

type AddRoomBlacklistRequest struct {
	ArtifactID string `json:"artifact_id,omitempty"`
	GroupName  string `json:"group_name,omitempty"`
}

type ArtifactSearchResult struct {
	Chunk          ArtifactChunk
	Artifact       Artifact
	RelevanceScore float64
}

type ArtifactRepository interface {
	CreateGroup(ctx context.Context, group *ArtifactGroup) error
	GetGroupByName(ctx context.Context, projectID string, name string) (*ArtifactGroup, error)
	ListGroups(ctx context.Context, projectID string) ([]ArtifactGroup, error)

	CreateArtifact(ctx context.Context, artifact *Artifact) error
	UpdateArtifact(ctx context.Context, artifact *Artifact) error
	GetArtifactByID(ctx context.Context, artifactID string) (*Artifact, error)
	GetArtifactByPath(ctx context.Context, projectID string, filePath string) (*Artifact, error)
	ListArtifacts(ctx context.Context, projectID string, groupName string, limit int, offset int) ([]Artifact, int, error)
	DeleteArtifact(ctx context.Context, artifactID string) error

	CreateChunks(ctx context.Context, chunks []ArtifactChunk) error
	DeleteChunksByArtifact(ctx context.Context, artifactID string) error
	SearchChunksVector(ctx context.Context, projectID string, queryEmbedding []float32, limit int, excludedArtifactIDs []string, excludedGroupNames []string) ([]ArtifactSearchResult, error)
	ListPendingChunks(ctx context.Context, limit int) ([]ArtifactChunk, error)
	UpsertChunkVectors(ctx context.Context, chunks []ArtifactChunk, groupName string) error
	MarkChunksIndexed(ctx context.Context, chunkIDs []string) error

	AddReferences(ctx context.Context, sourceArtifactID string, targetArtifactIDs []string, relationshipType string) error
	GetOutgoingReferences(ctx context.Context, sourceArtifactID string) ([]ArtifactReference, error)
	GetKnowledgeGraph(ctx context.Context, projectID string) (*KnowledgeGraph, error)
	DeleteReferencesBySource(ctx context.Context, sourceArtifactID string) error

	AddRoomBlacklist(ctx context.Context, blacklist *RoomArtifactBlacklist) error
	RemoveRoomBlacklist(ctx context.Context, blacklistID string) error
	GetRoomBlacklists(ctx context.Context, roomID string) ([]RoomArtifactBlacklist, error)
}

type ArtifactUsecase interface {
	IngestArtifact(ctx context.Context, projectID string, req CreateArtifactRequest) (*Artifact, error)
	UpdateArtifact(ctx context.Context, projectID string, artifactID string, req UpdateArtifactRequest) (*Artifact, error)
	GetArtifact(ctx context.Context, projectID string, artifactID string) (*Artifact, error)
	ListArtifacts(ctx context.Context, projectID string, groupName string, limit int, offset int) ([]Artifact, int, error)
	DeleteArtifact(ctx context.Context, projectID string, artifactID string) error
	GetKnowledgeGraph(ctx context.Context, projectID string) (*KnowledgeGraph, error)

	AddRoomBlacklist(ctx context.Context, projectID string, roomID string, req AddRoomBlacklistRequest) (*RoomArtifactBlacklist, error)
	RemoveRoomBlacklist(ctx context.Context, projectID string, roomID string, blacklistID string) error
	GetRoomBlacklists(ctx context.Context, projectID string, roomID string) ([]RoomArtifactBlacklist, error)

	RetrieveArtifactContext(ctx context.Context, projectID string, roomID string, query string, policy ContextPolicy) ([]ArtifactContext, error)
	RetryPendingEmbeds(ctx context.Context, limit int) (int, error)
}
