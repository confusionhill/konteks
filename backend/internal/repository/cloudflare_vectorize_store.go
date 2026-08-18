package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	"github.com/dika/contextkit/backend/internal/config"
	"github.com/dika/contextkit/backend/internal/domain"
)

type cloudflareVectorizeStore struct {
	accountID     string
	apiToken      string
	memoryIndex   string
	artifactIndex string
	httpClient    *http.Client
}

// NewCloudflareVectorizeStore creates a VectorStore implementation using Cloudflare Vectorize REST API.
func NewCloudflareVectorizeStore(cfg *config.Config) domain.VectorStore {
	return &cloudflareVectorizeStore{
		accountID:     cfg.CloudflareAccountID,
		apiToken:      cfg.CloudflareAPIToken,
		memoryIndex:   cfg.CloudflareMemoryIndex,
		artifactIndex: cfg.CloudflareArtifactIndex,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type cfVector struct {
	ID       string         `json:"id"`
	Values   []float32      `json:"values"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type cfInsertRequest struct {
	Vectors []cfVector `json:"vectors"`
}

type cfQueryRequest struct {
	Vector         []float32      `json:"vector"`
	TopK           int            `json:"topK"`
	Filter         map[string]any `json:"filter,omitempty"`
	ReturnMetadata string         `json:"returnMetadata,omitempty"`
}

type cfQueryResult struct {
	ID    string  `json:"id"`
	Score float64 `json:"score"`
}

type cfResponse struct {
	Success bool            `json:"success"`
	Errors  []any           `json:"errors"`
	Result  json.RawMessage `json:"result"`
}

func (s *cloudflareVectorizeStore) doRequest(ctx context.Context, method, url string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("cloudflare vectorize API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var cfResp cfResponse
	if err := json.Unmarshal(respBytes, &cfResp); err != nil {
		return nil, err
	}

	if !cfResp.Success {
		return nil, fmt.Errorf("cloudflare vectorize API returned failure: %s", string(respBytes))
	}

	return cfResp.Result, nil
}

func (s *cloudflareVectorizeStore) UpsertMemoryVector(ctx context.Context, memory *domain.Memory) error {
	if len(memory.Embedding) == 0 {
		return nil
	}

	roomID := ""
	if memory.RoomID != nil {
		roomID = *memory.RoomID
	}
	extUserID := ""
	if memory.ExternalUserID != nil {
		extUserID = *memory.ExternalUserID
	}

	vec := cfVector{
		ID:     memory.ID,
		Values: memory.Embedding,
		Metadata: map[string]any{
			"project_id":       memory.ProjectID,
			"room_id":          roomID,
			"external_user_id": extUserID,
			"status":           "active",
		},
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/vectorize/v2/indexes/%s/insert", s.accountID, s.memoryIndex)
	_, err := s.doRequest(ctx, http.MethodPost, url, cfInsertRequest{Vectors: []cfVector{vec}})
	return err
}

// UpdateMemoryMetadata updates the stored metadata for a single memory vector.
// The v2 API has no metadata-only endpoint, so we fetch the existing vector
// (which includes its values) and re-upsert it with the merged metadata.
func (s *cloudflareVectorizeStore) UpdateMemoryMetadata(ctx context.Context, memoryID string, metadata map[string]any) error {
	getURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/vectorize/v2/indexes/%s/get_by_ids", s.accountID, s.memoryIndex)
	resBytes, err := s.doRequest(ctx, http.MethodPost, getURL, map[string]any{"ids": []string{memoryID}})
	if err != nil {
		return err
	}

	var existing []cfVector
	if err := json.Unmarshal(resBytes, &existing); err != nil {
		return err
	}
	if len(existing) == 0 {
		return fmt.Errorf("vector not found for memory %s", memoryID)
	}

	vec := existing[0]
	if len(vec.Values) == 0 {
		return fmt.Errorf("vector %s has no values to upsert", memoryID)
	}

	merged := make(map[string]any, len(vec.Metadata)+len(metadata))
	for k, v := range vec.Metadata {
		merged[k] = v
	}
	for k, v := range metadata {
		merged[k] = v
	}
	vec.Metadata = merged

	line, err := json.Marshal(vec)
	if err != nil {
		return err
	}
	ndjson := append(line, '\n')

	upsertURL := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/vectorize/v2/indexes/%s/upsert", s.accountID, s.memoryIndex)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, upsertURL, bytes.NewReader(ndjson))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/x-ndjson")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("cloudflare vectorize upsert error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var cfResp cfResponse
	if err := json.Unmarshal(respBody, &cfResp); err != nil {
		return err
	}
	if !cfResp.Success {
		return fmt.Errorf("cloudflare vectorize upsert returned failure: %s", string(respBody))
	}
	return nil
}

func (s *cloudflareVectorizeStore) queryMemoryIndex(ctx context.Context, queryEmbedding []float32, limit int, filter map[string]any) ([]domain.VectorMatch, error) {
	reqBody := cfQueryRequest{
		Vector: queryEmbedding,
		TopK:   limit,
		Filter: filter,
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/vectorize/v2/indexes/%s/query", s.accountID, s.memoryIndex)
	resBytes, err := s.doRequest(ctx, http.MethodPost, url, reqBody)
	if err != nil {
		return nil, err
	}

	type queryResultContainer struct {
		Matches []cfQueryResult `json:"matches"`
	}

	var res queryResultContainer
	if err := json.Unmarshal(resBytes, &res); err != nil {
		return nil, err
	}

	matches := make([]domain.VectorMatch, len(res.Matches))
	for i, m := range res.Matches {
		matches[i] = domain.VectorMatch{
			ID:    m.ID,
			Score: m.Score,
		}
	}

	return matches, nil
}

// SearchMemoryVectors finds the nearest memory vectors for the query embedding.
// The metadata filter narrows results to the given project, and (when known) to
// the room plus any memories belonging to the external user. Cloudflare
// Vectorize only supports AND-style filters, so OR semantics are implemented by
// running one query per clause and merging the best match per vector ID.
func (s *cloudflareVectorizeStore) SearchMemoryVectors(ctx context.Context, projectID string, roomID string, externalUserID *string, queryEmbedding []float32, limit int) ([]domain.VectorMatch, error) {
	base := map[string]any{
		"project_id": projectID,
		"status":     "active",
	}

	copyMap := func(m map[string]any, extraKey string, extraVal string) map[string]any {
		c := make(map[string]any, len(m)+1)
		for k, v := range m {
			c[k] = v
		}
		c[extraKey] = extraVal
		return c
	}

	roomFilter := copyMap(base, "room_id", roomID)

	allMatches := []domain.VectorMatch{}
	byID := map[string]float64{}

	appendMatches := func(matches []domain.VectorMatch) {
		for _, m := range matches {
			if score, ok := byID[m.ID]; ok {
				if m.Score > score {
					byID[m.ID] = m.Score
				}
				continue
			}
			byID[m.ID] = m.Score
			allMatches = append(allMatches, m)
		}
	}

	roomMatches, err := s.queryMemoryIndex(ctx, queryEmbedding, limit, roomFilter)
	if err != nil {
		return nil, fmt.Errorf("room memory query: %w", err)
	}
	appendMatches(roomMatches)

	if externalUserID != nil && *externalUserID != "" {
		userFilter := copyMap(base, "external_user_id", *externalUserID)

		userMatches, err := s.queryMemoryIndex(ctx, queryEmbedding, limit, userFilter)
		if err != nil {
			return nil, fmt.Errorf("user memory query: %w", err)
		}
		appendMatches(userMatches)
	}

	sort.Slice(allMatches, func(i, j int) bool {
		return allMatches[i].Score > allMatches[j].Score
	})

	if len(allMatches) > limit {
		allMatches = allMatches[:limit]
	}

	return allMatches, nil
}

type cfDeleteRequest struct {
	IDs []string `json:"ids"`
}

func (s *cloudflareVectorizeStore) DeleteMemoryVectors(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/vectorize/v2/indexes/%s/delete_by_ids", s.accountID, s.memoryIndex)
	_, err := s.doRequest(ctx, http.MethodPost, url, cfDeleteRequest{IDs: ids})
	return err
}

func (s *cloudflareVectorizeStore) UpsertArtifactChunkVectors(ctx context.Context, chunks []domain.ArtifactChunk, groupName string) error {
	if len(chunks) == 0 {
		return nil
	}

	vectors := make([]cfVector, 0, len(chunks))
	for _, chunk := range chunks {
		if len(chunk.Embedding) == 0 {
			continue
		}
		vectors = append(vectors, cfVector{
			ID:     chunk.ID,
			Values: chunk.Embedding,
			Metadata: map[string]any{
				"project_id":  chunk.ProjectID,
				"artifact_id": chunk.ArtifactID,
				"group_name":  groupName,
			},
		})
	}

	if len(vectors) == 0 {
		return nil
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/vectorize/v2/indexes/%s/insert", s.accountID, s.artifactIndex)
	_, err := s.doRequest(ctx, http.MethodPost, url, cfInsertRequest{Vectors: vectors})
	return err
}

func (s *cloudflareVectorizeStore) SearchArtifactChunkVectors(ctx context.Context, projectID string, queryEmbedding []float32, limit int, excludedArtifactIDs []string, excludedGroupNames []string) ([]domain.VectorMatch, error) {
	filter := map[string]any{
		"project_id": projectID,
	}

	reqBody := cfQueryRequest{
		Vector:         queryEmbedding,
		TopK:           limit,
		Filter:         filter,
		ReturnMetadata: "indexed",
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/vectorize/v2/indexes/%s/query", s.accountID, s.artifactIndex)
	resBytes, err := s.doRequest(ctx, http.MethodPost, url, reqBody)
	if err != nil {
		return nil, err
	}

	type cfMatchWithMeta struct {
		ID       string         `json:"id"`
		Score    float64        `json:"score"`
		Metadata map[string]any `json:"metadata"`
	}
	type queryResultContainer struct {
		Matches []cfMatchWithMeta `json:"matches"`
	}

	var res queryResultContainer
	if err := json.Unmarshal(resBytes, &res); err != nil {
		return nil, err
	}

	excludedArtMap := make(map[string]bool)
	for _, id := range excludedArtifactIDs {
		excludedArtMap[id] = true
	}
	excludedGrpMap := make(map[string]bool)
	for _, g := range excludedGroupNames {
		excludedGrpMap[g] = true
	}

	matches := make([]domain.VectorMatch, 0, len(res.Matches))
	for _, m := range res.Matches {
		if artID, ok := m.Metadata["artifact_id"].(string); ok && excludedArtMap[artID] {
			continue
		}
		if grpName, ok := m.Metadata["group_name"].(string); ok && grpName != "" && excludedGrpMap[grpName] {
			continue
		}
		matches = append(matches, domain.VectorMatch{
			ID:    m.ID,
			Score: m.Score,
		})
	}

	return matches, nil
}

func (s *cloudflareVectorizeStore) DeleteArtifactChunkVectors(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/vectorize/v2/indexes/%s/delete_by_ids", s.accountID, s.artifactIndex)
	_, err := s.doRequest(ctx, http.MethodPost, url, cfDeleteRequest{IDs: ids})
	return err
}
