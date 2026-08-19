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

type cloudflareD1Store struct {
	accountID  string
	apiToken   string
	httpClient *http.Client
}

// NewCloudflareD1Store creates a D1Store implementation using Cloudflare D1 REST API.
func NewCloudflareD1Store(cfg *config.Config) domain.D1Store {
	return &cloudflareD1Store{
		accountID: cfg.CloudflareAccountID,
		apiToken:  cfg.CloudflareAPIToken,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type d1QueryRequest struct {
	SQL    string `json:"sql"`
	Params []any  `json:"params,omitempty"`
}

type d1QueryResult struct {
	Results []map[string]any `json:"results"`
	Success bool             `json:"success"`
	Error   string           `json:"error,omitempty"`
}

type d1Response struct {
	Success bool            `json:"success"`
	Errors  []any           `json:"errors"`
	Result  []d1QueryResult `json:"result"`
}

func (s *cloudflareD1Store) doBatchRequest(ctx context.Context, databaseUUID string, reqBody any) ([]d1QueryResult, error) {
	if reqBody == nil {
		return nil, nil
	}

	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/d1/database/%s/query", s.accountID, databaseUUID)

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBytes))
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
		return nil, fmt.Errorf("cloudflare D1 API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var cfResp d1Response
	if err := json.Unmarshal(respBytes, &cfResp); err != nil {
		return nil, err
	}

	if !cfResp.Success {
		return nil, fmt.Errorf("cloudflare D1 API returned failure: %s", string(respBytes))
	}

	// Check inner results for errors
	for i, res := range cfResp.Result {
		if !res.Success {
			return nil, fmt.Errorf("cloudflare D1 API batch error at index %d: %s", i, res.Error)
		}
	}

	return cfResp.Result, nil
}

func (s *cloudflareD1Store) Query(ctx context.Context, databaseUUID string, sql string, args ...any) ([]map[string]any, error) {
	req := d1QueryRequest{
		SQL:    sql,
		Params: args,
	}

	results, err := s.doBatchRequest(ctx, databaseUUID, req)
	if err != nil {
		return nil, err
	}

	if len(results) == 0 {
		return []map[string]any{}, nil
	}
	return results[0].Results, nil
}

func (s *cloudflareD1Store) Batch(ctx context.Context, databaseUUID string, queries []string, args [][]any) ([][]map[string]any, error) {
	if len(queries) != len(args) && args != nil {
		return nil, fmt.Errorf("queries and args length mismatch")
	}

	var finalResults [][]map[string]any

	for i, q := range queries {
		var p []any
		if args != nil && len(args) > i {
			p = args[i]
		}
		
		req := d1QueryRequest{
			SQL:    q,
			Params: p,
		}

		results, err := s.doBatchRequest(ctx, databaseUUID, req)
		if err != nil {
			return nil, fmt.Errorf("batch query error at index %d: %w", i, err)
		}
		
		if len(results) > 0 {
			finalResults = append(finalResults, results[0].Results)
		} else {
			finalResults = append(finalResults, []map[string]any{})
		}
	}

	return finalResults, nil
}

type d1CreateDBRequest struct {
	Name string `json:"name"`
}

type d1CreateDBResponse struct {
	Success bool `json:"success"`
	Errors  []any `json:"errors"`
	Result  struct {
		UUID string `json:"uuid"`
		Name string `json:"name"`
	} `json:"result"`
}

func (s *cloudflareD1Store) CreateDatabase(ctx context.Context, name string) (string, error) {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/accounts/%s/d1/database", s.accountID)

	reqBody := d1CreateDBRequest{Name: name}
	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBytes))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+s.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("cloudflare D1 create DB error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var cfResp d1CreateDBResponse
	if err := json.Unmarshal(respBytes, &cfResp); err != nil {
		return "", err
	}

	if !cfResp.Success || cfResp.Result.UUID == "" {
		return "", fmt.Errorf("cloudflare D1 create DB failed: %s", string(respBytes))
	}

	return cfResp.Result.UUID, nil
}
