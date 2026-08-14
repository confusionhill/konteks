package usecase

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/dika/contextkit/backend/internal/domain"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type artifactUsecase struct {
	artifactRepo     domain.ArtifactRepository
	openRouterClient domain.OpenRouterServiceClient
	storageProvider  domain.StorageProvider
}

func NewArtifactUsecase(
	artifactRepo domain.ArtifactRepository,
	openRouterClient domain.OpenRouterServiceClient,
	storageProvider domain.StorageProvider,
) domain.ArtifactUsecase {
	return &artifactUsecase{
		artifactRepo:     artifactRepo,
		openRouterClient: openRouterClient,
		storageProvider:  storageProvider,
	}
}

type FrontmatterHeader struct {
	Title      string   `yaml:"title"`
	Group      string   `yaml:"group"`
	Tags       []string `yaml:"tags"`
	References []string `yaml:"references"`
}

func parseFrontmatter(rawContent string) (FrontmatterHeader, string) {
	var header FrontmatterHeader
	trimmed := strings.TrimSpace(rawContent)
	if !strings.HasPrefix(trimmed, "---") {
		return header, rawContent
	}

	parts := strings.SplitN(trimmed[3:], "---", 2)
	if len(parts) < 2 {
		return header, rawContent
	}

	yamlBlock := parts[0]
	body := strings.TrimSpace(parts[1])

	_ = yaml.Unmarshal([]byte(yamlBlock), &header)
	return header, body
}

var (
	markdownLinkRegex = regexp.MustCompile(`\[.*?\]\(([^)]+\.md)(?:#[^)]+)?\)`)
	wikilinkRegex     = regexp.MustCompile(`\[\[(.*?)\]\]`)
)

func extractSyntaxReferences(content string) []string {
	refMap := make(map[string]bool)

	mdMatches := markdownLinkRegex.FindAllStringSubmatch(content, -1)
	for _, m := range mdMatches {
		if len(m) > 1 {
			refMap[strings.TrimSpace(m[1])] = true
		}
	}

	wikiMatches := wikilinkRegex.FindAllStringSubmatch(content, -1)
	for _, m := range wikiMatches {
		if len(m) > 1 {
			target := strings.TrimSpace(m[1])
			if !strings.HasSuffix(target, ".md") {
				target = target + ".md"
			}
			refMap[target] = true
		}
	}

	refs := make([]string, 0, len(refMap))
	for r := range refMap {
		refs = append(refs, r)
	}
	return refs
}

type HeaderChunk struct {
	SectionHeading string
	Content        string
}

func chunkMarkdownByHeaders(content string) []HeaderChunk {
	lines := strings.Split(content, "\n")
	var chunks []HeaderChunk

	var currentHeading = "General"
	var currentLines []string

	flush := func() {
		text := strings.TrimSpace(strings.Join(currentLines, "\n"))
		if text != "" {
			// Sub-chunk if section is huge (> 2000 chars)
			if len(text) > 2000 {
				paragraphs := strings.Split(text, "\n\n")
				var buf []string
				bufLen := 0
				for _, p := range paragraphs {
					if bufLen+len(p) > 1500 && len(buf) > 0 {
						chunks = append(chunks, HeaderChunk{
							SectionHeading: currentHeading,
							Content:        strings.TrimSpace(strings.Join(buf, "\n\n")),
						})
						buf = nil
						bufLen = 0
					}
					buf = append(buf, p)
					bufLen += len(p)
				}
				if len(buf) > 0 {
					chunks = append(chunks, HeaderChunk{
						SectionHeading: currentHeading,
						Content:        strings.TrimSpace(strings.Join(buf, "\n\n")),
					})
				}
			} else {
				chunks = append(chunks, HeaderChunk{
					SectionHeading: currentHeading,
					Content:        text,
				})
			}
		}
		currentLines = nil
	}

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") || strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### ") {
			flush()
			currentHeading = strings.TrimLeft(trimmed, "# ")
		} else {
			currentLines = append(currentLines, line)
		}
	}
	flush()

	if len(chunks) == 0 && strings.TrimSpace(content) != "" {
		chunks = append(chunks, HeaderChunk{
			SectionHeading: "General",
			Content:        strings.TrimSpace(content),
		})
	}
	return chunks
}

func (u *artifactUsecase) IngestArtifact(ctx context.Context, projectID string, req domain.CreateArtifactRequest) (*domain.Artifact, error) {
	if req.FilePath == "" {
		return nil, fmt.Errorf("file_path is required")
	}

	header, body := parseFrontmatter(req.Content)

	title := req.Title
	if title == "" {
		if header.Title != "" {
			title = header.Title
		} else {
			title = path.Base(req.FilePath)
		}
	}

	groupName := req.GroupName
	if groupName == "" && header.Group != "" {
		groupName = header.Group
	}

	hash := sha256.Sum256([]byte(req.Content))
	contentHash := hex.EncodeToString(hash[:])

	// Check if artifact with exact same content hash already exists
	existing, _ := u.artifactRepo.GetArtifactByPath(ctx, projectID, req.FilePath)
	if existing != nil && existing.ContentHash == contentHash {
		log.Printf("[Artifact Usecase]: Content hash unchanged for %s. Skipping re-indexing.", req.FilePath)
		return existing, nil
	}

	var groupID *string
	if groupName != "" {
		grp, errGrp := u.artifactRepo.GetGroupByName(ctx, projectID, groupName)
		if errGrp == nil && grp != nil {
			groupID = &grp.ID
		} else if grp == nil {
			newGrp := domain.ArtifactGroup{
				ProjectID:   projectID,
				Name:        groupName,
				FolderPath:  "/" + groupName,
				Description: "Auto-created group for " + groupName,
			}
			if errCreate := u.artifactRepo.CreateGroup(ctx, &newGrp); errCreate == nil {
				groupID = &newGrp.ID
			}
		}
	}

	artifactID := uuid.NewString()
	if existing != nil {
		artifactID = existing.ID
	}

	s3Key := fmt.Sprintf("projects/%s/artifacts/%s.md", projectID, artifactID)
	if u.storageProvider != nil {
		if _, errS3 := u.storageProvider.PutObject(ctx, s3Key, []byte(req.Content)); errS3 != nil {
			log.Printf("[Artifact Usecase Warning]: failed to upload .md to S3: %v", errS3)
		}
	}

	metaJSON := req.Metadata
	if len(metaJSON) == 0 && len(header.Tags) > 0 {
		metaMap := map[string]any{"tags": header.Tags}
		metaJSON, _ = json.Marshal(metaMap)
	}

	artifact := &domain.Artifact{
		ID:          artifactID,
		ProjectID:   projectID,
		GroupID:     groupID,
		GroupName:   groupName,
		FilePath:    req.FilePath,
		Title:       title,
		S3Key:       s3Key,
		ContentHash: contentHash,
		Metadata:    metaJSON,
		Content:     req.Content,
	}

	if existing != nil {
		if errUpd := u.artifactRepo.UpdateArtifact(ctx, artifact); errUpd != nil {
			return nil, errUpd
		}
		_ = u.artifactRepo.DeleteChunksByArtifact(ctx, artifactID)
		_ = u.artifactRepo.DeleteReferencesBySource(ctx, artifactID)
	} else {
		if errIns := u.artifactRepo.CreateArtifact(ctx, artifact); errIns != nil {
			return nil, errIns
		}
	}

	// Extract syntax links & frontmatter references
	syntaxRefs := extractSyntaxReferences(body)
	allRefsMap := make(map[string]bool)
	for _, r := range req.References {
		allRefsMap[r] = true
	}
	for _, r := range header.References {
		allRefsMap[r] = true
	}
	for _, r := range syntaxRefs {
		allRefsMap[r] = true
	}

	_ = u.artifactRepo.DeleteReferencesBySource(ctx, artifact.ID)

	var targetIDs []string
	for targetPath := range allRefsMap {
		targetArt, _ := u.artifactRepo.GetArtifactByPath(ctx, projectID, targetPath)
		if targetArt != nil && targetArt.ID != artifact.ID {
			targetIDs = append(targetIDs, targetArt.ID)
		}
	}
	if len(targetIDs) > 0 {
		_ = u.artifactRepo.AddReferences(ctx, artifact.ID, targetIDs, "refers_to")
	}

	// Bi-directional reference resolution: link existing artifacts that reference this file
	if allArts, _, errArts := u.artifactRepo.ListArtifacts(ctx, projectID, "", 100, 0); errArts == nil {
		baseName := path.Base(req.FilePath)
		for _, other := range allArts {
			if other.ID == artifact.ID {
				continue
			}
			if otherFull, errF := u.GetArtifact(ctx, projectID, other.ID); errF == nil && otherFull != nil && otherFull.Content != "" {
				_, otherBody := parseFrontmatter(otherFull.Content)
				otherRefs := extractSyntaxReferences(otherBody)
				for _, ref := range otherRefs {
					if ref == req.FilePath || ref == baseName || path.Base(ref) == baseName {
						_ = u.artifactRepo.AddReferences(ctx, other.ID, []string{artifact.ID}, "refers_to")
						break
					}
				}
			}
		}
	}

	// Asynchronous Batch Embedding Generation
	chunks := chunkMarkdownByHeaders(body)
	go u.asyncGenerateEmbeddingsAndSave(projectID, artifact.ID, chunks)

	artifact.ChunksCount = len(chunks)
	return artifact, nil
}

func (u *artifactUsecase) asyncGenerateEmbeddingsAndSave(projectID string, artifactID string, chunks []HeaderChunk) {
	bgCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var dbChunks []domain.ArtifactChunk
	targetDim := 0

	for i, c := range chunks {
		var emb []float32
		if u.openRouterClient != nil {
			chunkCtx, chunkCancel := context.WithTimeout(bgCtx, 30*time.Second)
			e, err := u.openRouterClient.GenerateEmbedding(chunkCtx, c.SectionHeading+"\n"+c.Content)
			chunkCancel()
			if err != nil {
				log.Printf("[Artifact Async Embedding Warning]: chunk %d embedding call failed: %v", i, err)
			} else {
				emb = e
			}
		}
		if len(emb) > 0 {
			targetDim = len(emb)
		} else if targetDim == 0 {
			targetDim = 4096
		}

		if len(emb) == 0 || len(emb) != targetDim {
			emb = make([]float32, targetDim)
		}

		dbChunks = append(dbChunks, domain.ArtifactChunk{
			ArtifactID:     artifactID,
			ProjectID:      projectID,
			ChunkIndex:     i,
			SectionHeading: c.SectionHeading,
			Content:        c.Content,
			Embedding:      emb,
		})
	}

	if err := u.artifactRepo.CreateChunks(bgCtx, dbChunks); err != nil {
		log.Printf("[Artifact Async Save Error]: failed saving chunks for artifact %s: %v", artifactID, err)
	} else {
		log.Printf("[Artifact Async Success]: indexed %d chunks for artifact %s (dim=%d)", len(dbChunks), artifactID, targetDim)
	}
}

func (u *artifactUsecase) UpdateArtifact(ctx context.Context, projectID string, artifactID string, req domain.UpdateArtifactRequest) (*domain.Artifact, error) {
	existing, err := u.artifactRepo.GetArtifactByID(ctx, artifactID)
	if err != nil || existing == nil {
		return nil, fmt.Errorf("artifact not found")
	}
	if existing.ProjectID != projectID {
		return nil, fmt.Errorf("unauthorized artifact project mismatch")
	}

	createReq := domain.CreateArtifactRequest{
		FilePath:   existing.FilePath,
		Title:      req.Title,
		Content:    req.Content,
		GroupName:  req.GroupName,
		References: req.References,
		Metadata:   req.Metadata,
	}
	return u.IngestArtifact(ctx, projectID, createReq)
}

func (u *artifactUsecase) GetArtifact(ctx context.Context, projectID string, artifactID string) (*domain.Artifact, error) {
	art, err := u.artifactRepo.GetArtifactByID(ctx, artifactID)
	if err != nil || art == nil {
		return nil, fmt.Errorf("artifact not found")
	}
	if art.ProjectID != projectID {
		return nil, fmt.Errorf("unauthorized")
	}

	if u.storageProvider != nil && art.S3Key != "" {
		contentBytes, errS3 := u.storageProvider.GetObject(ctx, art.S3Key)
		if errS3 == nil {
			art.Content = string(contentBytes)
		}
	}
	return art, nil
}

func (u *artifactUsecase) ListArtifacts(ctx context.Context, projectID string, groupName string, limit int, offset int) ([]domain.Artifact, int, error) {
	if limit <= 0 {
		limit = 50
	}
	return u.artifactRepo.ListArtifacts(ctx, projectID, groupName, limit, offset)
}

func (u *artifactUsecase) DeleteArtifact(ctx context.Context, projectID string, artifactID string) error {
	existing, err := u.artifactRepo.GetArtifactByID(ctx, artifactID)
	if err != nil || existing == nil {
		return fmt.Errorf("artifact not found")
	}
	if existing.ProjectID != projectID {
		return fmt.Errorf("unauthorized")
	}

	if u.storageProvider != nil && existing.S3Key != "" {
		_ = u.storageProvider.DeleteObject(ctx, existing.S3Key)
	}
	return u.artifactRepo.DeleteArtifact(ctx, artifactID)
}

func (u *artifactUsecase) GetKnowledgeGraph(ctx context.Context, projectID string) (*domain.KnowledgeGraph, error) {
	return u.artifactRepo.GetKnowledgeGraph(ctx, projectID)
}

func (u *artifactUsecase) AddRoomBlacklist(ctx context.Context, projectID string, roomID string, req domain.AddRoomBlacklistRequest) (*domain.RoomArtifactBlacklist, error) {
	if req.ArtifactID == "" && req.GroupName == "" {
		return nil, fmt.Errorf("either artifact_id or group_name must be provided")
	}

	var artID *string
	if req.ArtifactID != "" {
		artID = &req.ArtifactID
	}
	var grpName *string
	if req.GroupName != "" {
		grpName = &req.GroupName
	}

	bl := &domain.RoomArtifactBlacklist{
		RoomID:     roomID,
		ArtifactID: artID,
		GroupName:  grpName,
	}

	err := u.artifactRepo.AddRoomBlacklist(ctx, bl)
	if err != nil {
		return nil, err
	}
	return bl, nil
}

func (u *artifactUsecase) RemoveRoomBlacklist(ctx context.Context, projectID string, roomID string, blacklistID string) error {
	return u.artifactRepo.RemoveRoomBlacklist(ctx, blacklistID)
}

func (u *artifactUsecase) GetRoomBlacklists(ctx context.Context, projectID string, roomID string) ([]domain.RoomArtifactBlacklist, error) {
	return u.artifactRepo.GetRoomBlacklists(ctx, roomID)
}

var heuristicDocKeywords = []string{
	"how to", "spec", "documentation", ".md", "docs/", "setup", "config", "architecture", "policy", "schema", "rules",
}

func (u *artifactUsecase) RetrieveArtifactContext(ctx context.Context, projectID string, roomID string, query string, policy domain.ContextPolicy) ([]domain.ArtifactContext, error) {
	includeMode := strings.ToLower(strings.TrimSpace(policy.IncludeArtifacts))
	if includeMode == "" || includeMode == "false" {
		return nil, nil // Bypassed for sub-5ms turns
	}

	if includeMode == "auto" {
		isDocQuery := false
		qLower := strings.ToLower(query)
		for _, kw := range heuristicDocKeywords {
			if strings.Contains(qLower, kw) {
				isDocQuery = true
				break
			}
		}
		if !isDocQuery {
			return nil, nil // Heuristic bypassed
		}
	}

	// Fetch Room Blacklists
	blacklists, _ := u.artifactRepo.GetRoomBlacklists(ctx, roomID)
	var excludedArtIDs []string
	var excludedGroups []string

	for _, b := range blacklists {
		if b.ArtifactID != nil {
			excludedArtIDs = append(excludedArtIDs, *b.ArtifactID)
		}
		if b.GroupName != nil {
			excludedGroups = append(excludedGroups, *b.GroupName)
		}
	}
	for _, override := range policy.BlacklistOverride {
		if strings.Contains(override, "/") || strings.HasSuffix(override, ".md") {
			art, _ := u.artifactRepo.GetArtifactByPath(ctx, projectID, override)
			if art != nil {
				excludedArtIDs = append(excludedArtIDs, art.ID)
			}
		} else {
			excludedGroups = append(excludedGroups, override)
		}
	}

	// Embed query
	var queryEmbedding []float32
	if u.openRouterClient != nil && query != "" {
		emb, errEmb := u.openRouterClient.GenerateEmbedding(ctx, query)
		if errEmb == nil {
			queryEmbedding = emb
		}
	}
	if len(queryEmbedding) == 0 {
		queryEmbedding = make([]float32, 4096)
	}

	limit := 3
	searchResults, errSearch := u.artifactRepo.SearchChunksVector(ctx, projectID, queryEmbedding, limit, excludedArtIDs, excludedGroups)
	if errSearch != nil {
		log.Printf("[Artifact Context Error]: %v", errSearch)
		return nil, nil
	}

	if includeMode == "auto" && len(searchResults) > 0 && searchResults[0].RelevanceScore < 0.78 {
		log.Printf("[Artifact Auto Gating]: Top relevance %.2f < 0.78 threshold. Skipping artifacts.", searchResults[0].RelevanceScore)
		return nil, nil
	}

	tokenBudget := policy.MaxArtifactTokens
	if tokenBudget <= 0 {
		tokenBudget = 1000
	}

	var artifactContexts []domain.ArtifactContext
	visitedIDs := make(map[string]bool)

	for _, res := range searchResults {
		if visitedIDs[res.Artifact.ID] {
			continue
		}
		visitedIDs[res.Artifact.ID] = true

		artCtx := domain.ArtifactContext{
			ArtifactID:     res.Artifact.ID,
			FilePath:       res.Artifact.FilePath,
			Title:          res.Artifact.Title,
			SectionHeading: res.Chunk.SectionHeading,
			Content:        res.Chunk.Content,
			RelevanceScore: res.RelevanceScore,
		}

		// 1-Hop Graph Reference Expansion if enabled
		if policy.EnableGraphTraversal {
			refs, errRefs := u.artifactRepo.GetOutgoingReferences(ctx, res.Artifact.ID)
			if errRefs == nil && len(refs) > 0 {
				var graphRefs []domain.GraphReference
				for _, ref := range refs {
					targetArt, _ := u.artifactRepo.GetArtifactByID(ctx, ref.TargetArtifactID)
					if targetArt != nil {
						graphRefs = append(graphRefs, domain.GraphReference{
							ArtifactID:       targetArt.ID,
							FilePath:         targetArt.FilePath,
							Title:            targetArt.Title,
							RelationshipType: ref.RelationshipType,
						})
					}
				}
				artCtx.GraphReferences = graphRefs
			}
		}

		artifactContexts = append(artifactContexts, artCtx)
	}

	return artifactContexts, nil
}
