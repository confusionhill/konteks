package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	ReturnMetadata bool           `json:"returnMetadata"`
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

func (s *cloudflareVectorizeStore) SearchMemoryVectors(ctx context.Context, projectID string, roomID string, externalUserID *string, queryEmbedding []float32, limit int) ([]domain.VectorMatch, error) {
	filter := map[string]any{
		"project_id": projectID,
		"status":     "active",
	}

	if externalUserID != nil && *externalUserID != "" {
		filter["$or"] = []map[string]any{
			{"room_id": roomID},
			{"external_user_id": *externalUserID},
		}
	} else {
		filter["room_id"] = roomID
	}

	reqBody := cfQueryRequest{
		Vector:         queryEmbedding,
		TopK:           limit,
		Filter:         filter,
		ReturnMetadata: false,
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

func (s *cloudflareVectorizeStore) GetVectorByID(ctx context.Context, id string) ([]float32, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/vectorize/v2/indexes/%s/get_by_ids", s.accountID, s.memoryIndex)
	resBytes, err := s.doRequest(ctx, http.MethodPost, url, cfDeleteRequest{IDs: []string{id}})
	if err != nil {
		return nil, err
	}

	var vecs []cfVector
	if err := json.Unmarshal(resBytes, &vecs); err != nil {
		return nil, err
	}

	if len(vecs) == 0 {
		return nil, fmt.Errorf("vector id %s not found in cloudflare vectorize", id)
	}

	return vecs[0].Values, nil
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
		ReturnMetadata: true,
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
