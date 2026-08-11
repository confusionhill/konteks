package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dika/contextkit/backend/internal/config"
	"github.com/dika/contextkit/backend/internal/domain"
)

type openRouterServiceClient struct {
	cfg    *config.Config
	client *http.Client
}

func NewOpenRouterServiceClient(cfg *config.Config) domain.OpenRouterServiceClient {
	return &openRouterServiceClient{
		cfg:    cfg,
		client: &http.Client{},
	}
}

type embeddingRequest struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (s *openRouterServiceClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	url := fmt.Sprintf("%s/embeddings", strings.TrimSuffix(s.cfg.OpenRouterBaseURL, "/"))
	reqBody := embeddingRequest{
		Model: s.cfg.OpenRouterEmbeddingModel,
		Input: text,
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.cfg.OpenRouterAPIKey))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter embedding api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var respBody embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, err
	}

	if len(respBody.Data) == 0 {
		return nil, fmt.Errorf("openrouter embedding api returned empty data")
	}

	return respBody.Data[0].Embedding, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model          string        `json:"model"`
	Messages       []chatMessage `json:"messages"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

type extractionJSON struct {
	Summary     string          `json:"summary"`
	ActiveTopic string          `json:"active_topic"`
	State       json.RawMessage `json:"state"`
}

func (s *openRouterServiceClient) ExtractStateAndSummary(ctx context.Context, history []domain.Message, currentSummary string) (string, string, string, error) {
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(s.cfg.OpenRouterBaseURL, "/"))

	systemPrompt := `You are a conversation state and summary extraction engine. Analyze the conversation history and the current summary to produce an updated state package.
The output must be a valid JSON object matching exactly this schema:
{
  "summary": "a concise description of the conversation progress",
  "active_topic": "the primary active topic (e.g. hotel search, flight booking, payment)",
  "state": {}
}

Guidelines:
1. 'summary': Keep it concise but ensure it captures context (e.g., "User is comparing budget hotels in Bali").
2. 'active_topic': Briefly name the active user intent.
3. 'state': A flat JSON object containing key-value pairs representing parameters discussed so far (e.g., budget, location, date, amenities). Retain existing state parameters unless they are explicitly updated, refuted, or changed in the new turn.`

	// Formulate history string
	var histBuilder strings.Builder
	histBuilder.WriteString("Current Summary: " + currentSummary + "\n\nConversation History:\n")
	for _, m := range history {
		histBuilder.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
	}

	reqBody := chatCompletionRequest{
		Model: s.cfg.OpenRouterChatModel,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: histBuilder.String()},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", "", "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.cfg.OpenRouterAPIKey))

	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", "", "", fmt.Errorf("openrouter chat completion api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var respBody chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", "", "", err
	}

	if len(respBody.Choices) == 0 {
		return "", "", "", fmt.Errorf("openrouter chat completion api returned empty choices")
	}

	content := respBody.Choices[0].Message.Content

	var extraction extractionJSON
	if err := json.Unmarshal([]byte(content), &extraction); err != nil {
		return "", "", "", fmt.Errorf("failed to parse JSON extraction response: %w. raw content: %s", err, content)
	}

	stateStr := string(extraction.State)
	if stateStr == "" || stateStr == "null" {
		stateStr = "{}"
	}

	return extraction.Summary, extraction.ActiveTopic, stateStr, nil
}
