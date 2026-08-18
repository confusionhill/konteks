package usecase

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
)

type mockArtifactRepo struct {
	artifacts  map[string]*domain.Artifact
	groups     map[string]*domain.ArtifactGroup
	chunks     map[string][]domain.ArtifactChunk
	refs       map[string][]domain.ArtifactReference
	blacklists map[string][]domain.RoomArtifactBlacklist
	scores     map[string]float64
}

func newMockArtifactRepo() *mockArtifactRepo {
	return &mockArtifactRepo{
		artifacts:  make(map[string]*domain.Artifact),
		groups:     make(map[string]*domain.ArtifactGroup),
		chunks:     make(map[string][]domain.ArtifactChunk),
		refs:       make(map[string][]domain.ArtifactReference),
		blacklists: make(map[string][]domain.RoomArtifactBlacklist),
		scores:     make(map[string]float64),
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
func (m *mockArtifactRepo) ListPendingChunks(ctx context.Context, limit int) ([]domain.ArtifactChunk, error) {
	var pending []domain.ArtifactChunk
	for _, list := range m.chunks {
		for _, c := range list {
			if c.VectorStatus == domain.ChunkVectorStatusPending {
				pending = append(pending, c)
			}
		}
	}
	return pending, nil
}
func (m *mockArtifactRepo) UpsertChunkVectors(ctx context.Context, chunks []domain.ArtifactChunk, groupName string) error {
	for _, c := range chunks {
		list := m.chunks[c.ArtifactID]
		for i := range list {
			if list[i].ID == c.ID {
				list[i].Embedding = c.Embedding
				list[i].VectorStatus = c.VectorStatus
			}
		}
	}
	return nil
}
func (m *mockArtifactRepo) MarkChunksIndexed(ctx context.Context, chunkIDs []string) error {
	for _, list := range m.chunks {
		for i := range list {
			for _, id := range chunkIDs {
				if list[i].ID == id {
					list[i].VectorStatus = domain.ChunkVectorStatusIndexed
				}
			}
		}
	}
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
			score, ok := m.scores[c.ID]
			if !ok {
				score = 0.85
			}
			results = append(results, domain.ArtifactSearchResult{
				Chunk:          c,
				Artifact:       *art,
				RelevanceScore: score,
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
	usecase := NewArtifactUsecase(repo, &mockOpenRouter{}, storage)

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

// flakyOpenRouter fails GenerateEmbedding for the first n calls then succeeds,
// mimicking a transient embedding API outage.
type flakyOpenRouter struct {
	openRouterClient  *mockOpenRouter
	failures          int
	emptyEmbeddingFor int
	calls             int
	recordedDim       []int
}

func (f *flakyOpenRouter) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	f.calls++
	if f.calls <= f.failures {
		return nil, fmt.Errorf("simulated embedding outage")
	}
	if f.emptyEmbeddingFor > 0 && f.calls%2 == 0 {
		return []float32{}, nil
	}
	emb, err := f.openRouterClient.GenerateEmbedding(ctx, text)
	if err == nil {
		f.recordedDim = append(f.recordedDim, len(emb))
	}
	return emb, err
}
func (f *flakyOpenRouter) ExtractStateAndSummary(ctx context.Context, history []domain.Message, currentSummary string, currentStateJSON string, schemaJSON string) (string, string, string, []string, error) {
	return f.openRouterClient.ExtractStateAndSummary(ctx, history, currentSummary, currentStateJSON, schemaJSON)
}
func (f *flakyOpenRouter) EvaluateContext(ctx context.Context, query string, contextPkg *domain.ContextPackage) (*domain.ContextEvaluation, error) {
	return f.openRouterClient.EvaluateContext(ctx, query, contextPkg)
}
func (f *flakyOpenRouter) EvaluateContradiction(ctx context.Context, factA string, factB string) (bool, float64, error) {
	return f.openRouterClient.EvaluateContradiction(ctx, factA, factB)
}

func waitForChunks(t *testing.T, repo *mockArtifactRepo, artifactID string, want int) []domain.ArtifactChunk {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if chunks := repo.chunks[artifactID]; len(chunks) >= want {
			return chunks
		}
		time.Sleep(10 * time.Millisecond)
	}
	return repo.chunks[artifactID]
}

func TestArtifactIngestWithEmbeddingFailureSkipsVector(t *testing.T) {
	repo := newMockArtifactRepo()
	flaky := &flakyOpenRouter{openRouterClient: &mockOpenRouter{}, failures: 1}
	usecase := NewArtifactUsecase(repo, flaky, &mockStorage{})

	req := domain.CreateArtifactRequest{
		FilePath: "docs/ops.md",
		Title:    "Ops Runbook",
		Content:  "# Runbook\nStep one.\n\n# Backup\nStep two.",
	}
	art, err := usecase.IngestArtifact(context.Background(), "proj_1", req)
	if err != nil {
		t.Fatalf("unexpected error ingesting artifact: %v", err)
	}

	chunks := waitForChunks(t, repo, art.ID, 2)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}

	if chunks[0].VectorStatus == domain.ChunkVectorStatusIndexed {
		t.Errorf("expected chunk 0 pending after embedding failure, got %s", chunks[0].VectorStatus)
	}
	if len(chunks[0].Embedding) != 0 {
		t.Errorf("expected chunk 0 to have no embedding, got %d dims", len(chunks[0].Embedding))
	}
	if chunks[0].Content == "" {
		t.Errorf("expected chunk 0 text to be preserved in PG")
	}
	for i := 1; i < len(chunks); i++ {
		if chunks[i].VectorStatus != domain.ChunkVectorStatusIndexed {
			t.Errorf("expected chunk %d indexed, got %s", i, chunks[i].VectorStatus)
		}
		if len(chunks[i].Embedding) == 0 {
			t.Errorf("expected chunk %d to carry an embedding", i)
		}
	}

	for _, dim := range flaky.recordedDim {
		if dim != 2 {
			t.Errorf("expected embeddings to match model dim 2, got %d", dim)
		}
	}
}

func TestArtifactRetryPendingEmbeds(t *testing.T) {
	repo := newMockArtifactRepo()
	flaky := &flakyOpenRouter{openRouterClient: &mockOpenRouter{}, failures: 1}
	usecase := NewArtifactUsecase(repo, flaky, &mockStorage{})

	req := domain.CreateArtifactRequest{
		FilePath: "docs/retry.md",
		Title:    "Retry Doc",
		Content:  "# Runbook\nStep one.\n\n# Backup\nStep two.",
	}
	art, err := usecase.IngestArtifact(context.Background(), "proj_1", req)
	if err != nil {
		t.Fatalf("unexpected error ingesting artifact: %v", err)
	}

	chunks := waitForChunks(t, repo, art.ID, 2)
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d", len(chunks))
	}
	if chunks[0].VectorStatus != domain.ChunkVectorStatusPending {
		t.Fatalf("expected chunk 0 pending before retry, got %s", chunks[0].VectorStatus)
	}

	retried, errRet := usecase.RetryPendingEmbeds(context.Background(), 10)
	if errRet != nil {
		t.Fatalf("unexpected retry error: %v", errRet)
	}
	if retried != 1 {
		t.Errorf("expected 1 chunk retried, got %d", retried)
	}

	chunks = repo.chunks[art.ID]
	if chunks[0].VectorStatus != domain.ChunkVectorStatusIndexed {
		t.Errorf("expected chunk 0 indexed after retry, got %s", chunks[0].VectorStatus)
	}
	if len(chunks[0].Embedding) != 2 {
		t.Errorf("expected re-embedded chunk 0 embedding pushed to store, got %d dims", len(chunks[0].Embedding))
	}
}

func TestRetrieveArtifactContextFiltersLowRelevance(t *testing.T) {
	repo := newMockArtifactRepo()
	storage := &mockStorage{}
	usecase := NewArtifactUsecase(repo, &mockOpenRouter{}, storage)

	projectID := "proj_1"
	req := domain.CreateArtifactRequest{
		FilePath: "docs/policy.md",
		Title:    "Travel Policy",
		Content:  "# Travel Policy\nRules here.",
	}
	art, err := usecase.IngestArtifact(context.Background(), projectID, req)
	if err != nil {
		t.Fatalf("unexpected error ingesting artifact: %v", err)
	}
	chunks := waitForChunks(t, repo, art.ID, 1)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	// Strong match survives the default 0.5 floor.
	repo.scores[chunks[0].ID] = 0.62
	policyTrue := domain.ContextPolicy{IncludeArtifacts: "true"}

	artCtxs, errRet := usecase.RetrieveArtifactContext(context.Background(), projectID, "room_1", "travel policy refunds", policyTrue)
	if errRet != nil {
		t.Fatalf("unexpected error retrieving context: %v", errRet)
	}
	if len(artCtxs) != 1 {
		t.Fatalf("expected 1 artifact to pass 0.5 floor, got %d", len(artCtxs))
	}
	if artCtxs[0].FilePath != "docs/policy.md" {
		t.Errorf("expected docs/policy.md, got %s", artCtxs[0].FilePath)
	}
}

func TestRetrieveArtifactContextEmptyBelowFloor(t *testing.T) {
	repo := newMockArtifactRepo()
	storage := &mockStorage{}
	usecase := NewArtifactUsecase(repo, &mockOpenRouter{}, storage)

	projectID := "proj_1"
	req := domain.CreateArtifactRequest{
		FilePath: "docs/tokyo.md",
		Title:    "Destination Guide Tokyo",
		Content:  "# Tokyo Guide\nSights here.",
	}
	art, err := usecase.IngestArtifact(context.Background(), projectID, req)
	if err != nil {
		t.Fatalf("unexpected error ingesting artifact: %v", err)
	}
	chunks := waitForChunks(t, repo, art.ID, 1)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	repo.scores[chunks[0].ID] = 0.43 // Below the default 0.5 floor.
	policyTrue := domain.ContextPolicy{IncludeArtifacts: "true"}

	artCtxs, errRet := usecase.RetrieveArtifactContext(context.Background(), projectID, "room_1", "travel policy refunds", policyTrue)
	if errRet != nil {
		t.Fatalf("unexpected error retrieving context: %v", errRet)
	}
	if len(artCtxs) != 0 {
		t.Fatalf("expected 0 artifacts below floor, got %d", len(artCtxs))
	}
}

func TestRetrieveArtifactContextFloorOverriddenByPolicy(t *testing.T) {
	repo := newMockArtifactRepo()
	storage := &mockStorage{}
	usecase := NewArtifactUsecase(repo, &mockOpenRouter{}, storage)

	projectID := "proj_1"
	req := domain.CreateArtifactRequest{
		FilePath: "docs/vip.md",
		Title:    "Vip Guest Preferences",
		Content:  "# VIP Preferences\nLoyalty perks here.",
	}
	art, err := usecase.IngestArtifact(context.Background(), projectID, req)
	if err != nil {
		t.Fatalf("unexpected error ingesting artifact: %v", err)
	}
	chunks := waitForChunks(t, repo, art.ID, 1)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	// Custom min_relevance_score (0.63) keeps a 0.62 match out even in forced mode.
	repo.scores[chunks[0].ID] = 0.62
	policy := domain.ContextPolicy{IncludeArtifacts: "true", MinRelevanceScore: 0.63}

	artCtxs, errRet := usecase.RetrieveArtifactContext(context.Background(), projectID, "room_1", "vip perks", policy)
	if errRet != nil {
		t.Fatalf("unexpected error retrieving context: %v", errRet)
	}
	if len(artCtxs) != 0 {
		t.Fatalf("expected 0 artifacts when policy floor 0.63 > match 0.62, got %d", len(artCtxs))
	}
}

func TestRetrieveArtifactContextAutoModeFloor(t *testing.T) {
	repo := newMockArtifactRepo()
	storage := &mockStorage{}
	usecase := NewArtifactUsecase(repo, &mockOpenRouter{}, storage)

	projectID := "proj_1"
	req := domain.CreateArtifactRequest{
		FilePath: "docs/ops.md",
		Title:    "Ops Runbook",
		Content:  "# Ops Runbook\nDeploy steps here.",
	}
	art, err := usecase.IngestArtifact(context.Background(), projectID, req)
	if err != nil {
		t.Fatalf("unexpected error ingesting artifact: %v", err)
	}
	chunks := waitForChunks(t, repo, art.ID, 1)
	if len(chunks) != 1 {
		t.Fatalf("expected 1 chunk, got %d", len(chunks))
	}

	// Auto mode: 0.80 passes both the 0.78 top gate and the 0.5 floor.
	repo.scores[chunks[0].ID] = 0.80
	policyAuto := domain.ContextPolicy{IncludeArtifacts: "auto"}

	artCtxs, errRet := usecase.RetrieveArtifactContext(context.Background(), projectID, "room_1", "how to setup", policyAuto)
	if errRet != nil {
		t.Fatalf("unexpected error retrieving context: %v", errRet)
	}
	if len(artCtxs) != 1 {
		t.Fatalf("expected 1 artifact in auto mode, got %d", len(artCtxs))
	}

	// Auto mode: 0.60 clears the floor but fails the 0.78 top gate -> skipped.
	repo.scores[chunks[0].ID] = 0.60
	artCtxs, errRet = usecase.RetrieveArtifactContext(context.Background(), projectID, "room_1", "how to setup v2", policyAuto)
	if errRet != nil {
		t.Fatalf("unexpected error retrieving context: %v", errRet)
	}
	if len(artCtxs) != 0 {
		t.Fatalf("expected 0 artifacts below auto 0.78 top gate, got %d", len(artCtxs))
	}
}
