package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
)

type mockArtifactRepo struct {
	artifacts map[string]*domain.Artifact
	groups    map[string]*domain.ArtifactGroup
	chunks    map[string][]domain.ArtifactChunk
	refs      map[string][]domain.ArtifactReference
	blacklists map[string][]domain.RoomArtifactBlacklist
}

func newMockArtifactRepo() *mockArtifactRepo {
	return &mockArtifactRepo{
		artifacts:  make(map[string]*domain.Artifact),
		groups:     make(map[string]*domain.ArtifactGroup),
		chunks:     make(map[string][]domain.ArtifactChunk),
		refs:       make(map[string][]domain.ArtifactReference),
		blacklists: make(map[string][]domain.RoomArtifactBlacklist),
	}
}

func (m *mockArtifactRepo) CreateGroup(ctx context.Context, group *domain.ArtifactGroup) error {
	group.ID = "grp_" + group.Name
	m.groups[group.Name] = group
	return nil
}
func (m *mockArtifactRepo) GetGroupByName(ctx context.Context, projectID string, name string) (*domain.ArtifactGroup, error) {
	return m.groups[name], nil
}
func (m *mockArtifactRepo) ListGroups(ctx context.Context, projectID string) ([]domain.ArtifactGroup, error) {
	var res []domain.ArtifactGroup
	for _, g := range m.groups {
		res = append(res, *g)
	}
	return res, nil
}
func (m *mockArtifactRepo) CreateArtifact(ctx context.Context, artifact *domain.Artifact) error {
	m.artifacts[artifact.ID] = artifact
	return nil
}
func (m *mockArtifactRepo) UpdateArtifact(ctx context.Context, artifact *domain.Artifact) error {
	m.artifacts[artifact.ID] = artifact
	return nil
}
func (m *mockArtifactRepo) GetArtifactByID(ctx context.Context, artifactID string) (*domain.Artifact, error) {
	return m.artifacts[artifactID], nil
}
func (m *mockArtifactRepo) GetArtifactByPath(ctx context.Context, projectID string, filePath string) (*domain.Artifact, error) {
	for _, a := range m.artifacts {
		if a.ProjectID == projectID && a.FilePath == filePath {
			return a, nil
		}
	}
	return nil, nil
}
func (m *mockArtifactRepo) ListArtifacts(ctx context.Context, projectID string, groupName string, limit int, offset int) ([]domain.Artifact, int, error) {
	var res []domain.Artifact
	for _, a := range m.artifacts {
		if a.ProjectID == projectID {
			if groupName == "" || a.GroupName == groupName {
				res = append(res, *a)
			}
		}
	}
	return res, len(res), nil
}
func (m *mockArtifactRepo) DeleteArtifact(ctx context.Context, artifactID string) error {
	delete(m.artifacts, artifactID)
	return nil
}
func (m *mockArtifactRepo) CreateChunks(ctx context.Context, chunks []domain.ArtifactChunk) error {
	if len(chunks) > 0 {
		artID := chunks[0].ArtifactID
		m.chunks[artID] = chunks
	}
	return nil
}
func (m *mockArtifactRepo) DeleteChunksByArtifact(ctx context.Context, artifactID string) error {
	delete(m.chunks, artifactID)
	return nil
}
func (m *mockArtifactRepo) SearchChunksVector(ctx context.Context, projectID string, queryEmbedding []float32, limit int, excludedArtifactIDs []string, excludedGroupNames []string) ([]domain.ArtifactSearchResult, error) {
	var results []domain.ArtifactSearchResult
	excludedArtMap := make(map[string]bool)
	for _, id := range excludedArtifactIDs {
		excludedArtMap[id] = true
	}
	excludedGrpMap := make(map[string]bool)
	for _, g := range excludedGroupNames {
		excludedGrpMap[g] = true
	}

	for _, art := range m.artifacts {
		if art.ProjectID != projectID || excludedArtMap[art.ID] || excludedGrpMap[art.GroupName] {
			continue
		}
		chunks := m.chunks[art.ID]
		for _, c := range chunks {
			results = append(results, domain.ArtifactSearchResult{
				Chunk:          c,
				Artifact:       *art,
				RelevanceScore: 0.85,
			})
		}
	}
	return results, nil
}
func (m *mockArtifactRepo) AddReferences(ctx context.Context, sourceArtifactID string, targetArtifactIDs []string, relationshipType string) error {
	var refs []domain.ArtifactReference
	for _, t := range targetArtifactIDs {
		refs = append(refs, domain.ArtifactReference{
			SourceArtifactID: sourceArtifactID,
			TargetArtifactID: t,
			RelationshipType: relationshipType,
		})
	}
	m.refs[sourceArtifactID] = refs
	return nil
}
func (m *mockArtifactRepo) GetOutgoingReferences(ctx context.Context, sourceArtifactID string) ([]domain.ArtifactReference, error) {
	return m.refs[sourceArtifactID], nil
}
func (m *mockArtifactRepo) GetKnowledgeGraph(ctx context.Context, projectID string) (*domain.KnowledgeGraph, error) {
	var nodes []domain.GraphNode
	for _, a := range m.artifacts {
		if a.ProjectID == projectID {
			nodes = append(nodes, domain.GraphNode{
				ID:        a.ID,
				FilePath:  a.FilePath,
				Title:     a.Title,
				GroupName: a.GroupName,
			})
		}
	}
	var edges []domain.GraphEdge
	for src, refList := range m.refs {
		for _, r := range refList {
			edges = append(edges, domain.GraphEdge{
				Source:       src,
				Target:       r.TargetArtifactID,
				Relationship: r.RelationshipType,
			})
		}
	}
	return &domain.KnowledgeGraph{Nodes: nodes, Edges: edges}, nil
}
func (m *mockArtifactRepo) DeleteReferencesBySource(ctx context.Context, sourceArtifactID string) error {
	delete(m.refs, sourceArtifactID)
	return nil
}
func (m *mockArtifactRepo) AddRoomBlacklist(ctx context.Context, blacklist *domain.RoomArtifactBlacklist) error {
	blacklist.ID = "blk_1"
	m.blacklists[blacklist.RoomID] = append(m.blacklists[blacklist.RoomID], *blacklist)
	return nil
}
func (m *mockArtifactRepo) RemoveRoomBlacklist(ctx context.Context, blacklistID string) error {
	return nil
}
func (m *mockArtifactRepo) GetRoomBlacklists(ctx context.Context, roomID string) ([]domain.RoomArtifactBlacklist, error) {
	return m.blacklists[roomID], nil
}

func TestArtifactIngestionAndRetrieval(t *testing.T) {
	repo := newMockArtifactRepo()
	storage := &mockStorage{}
	usecase := NewArtifactUsecase(repo, nil, storage)

	projectID := "proj_1"
	req := domain.CreateArtifactRequest{
		FilePath: "docs/auth.md",
		Title:    "Auth Spec",
		Content:  "---\ngroup: security\n---\n# Auth Spec\nSee [Roles](docs/roles.md) for RBAC.",
	}

	art, err := usecase.IngestArtifact(context.Background(), projectID, req)
	if err != nil {
		t.Fatalf("unexpected error ingesting artifact: %v", err)
	}

	time.Sleep(100 * time.Millisecond) // Wait for async chunking worker to complete

	if art.FilePath != "docs/auth.md" {
		t.Errorf("expected file path docs/auth.md, got %s", art.FilePath)
	}
	if art.GroupName != "security" {
		t.Errorf("expected group_name security, got %s", art.GroupName)
	}

	// Test RetrieveArtifactContext with include_artifacts = "true"
	policy := domain.ContextPolicy{
		IncludeArtifacts:     "true",
		EnableGraphTraversal: true,
	}

	artCtxs, errRet := usecase.RetrieveArtifactContext(context.Background(), projectID, "room_1", "how to setup auth", policy)
	if errRet != nil {
		t.Fatalf("unexpected error retrieving context: %v", errRet)
	}

	if len(artCtxs) == 0 {
		t.Fatalf("expected at least 1 artifact context result")
	}
	if artCtxs[0].FilePath != "docs/auth.md" {
		t.Errorf("expected retrieved file_path docs/auth.md, got %s", artCtxs[0].FilePath)
	}
}
