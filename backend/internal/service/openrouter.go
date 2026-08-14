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
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
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

func (s *openRouterServiceClient) ExtractStateAndSummary(ctx context.Context, history []domain.Message, currentSummary string, schemaJSON string) (string, string, string, error) {
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

	if schemaJSON != "" && schemaJSON != "{}" && schemaJSON != "null" {
		systemPrompt += fmt.Sprintf("\n\nCRITICAL: The 'state' object in the output MUST adhere strictly to this schema: %s", schemaJSON)
	}

	// Formulate history string
	var histBuilder strings.Builder
	histBuilder.WriteString("Current Summary: " + currentSummary + "\n\nConversation History:\n")
	for _, m := range history {
		histBuilder.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
	}

	reqBody := chatCompletionRequest{
		Model: s.cfg.OpenRouterReasoningModel,
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

type referenceJSON struct {
	References map[string]string `json:"references"`
	Entities   []domain.Entity   `json:"entities"`
}

func (s *openRouterServiceClient) ResolveReferences(ctx context.Context, history []domain.Message, stateJSON string) (map[string]string, []domain.Entity, error) {
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(s.cfg.OpenRouterBaseURL, "/"))

	systemPrompt := `You are a Context as a Service Reference Resolver. Analyze recent conversation history and current room state to resolve ambiguous pronouns (e.g. 'it', 'that', 'the second option') to ground-truth entity names.

Rules:
1. ONLY resolve references and entities if they explicitly appear in the conversation history or room state.
2. If NO ambiguous pronouns or entities exist in the history, return empty structures: {"references": {}, "entities": []}.
3. NEVER copy example entities below unless they are explicitly mentioned in the provided conversation history.

JSON Schema format:
{
  "references": {
    "<pronoun_or_phrase>": "<resolved_entity_name>"
  },
  "entities": [
    {
      "type": "<entity_category>",
      "id": "<entity_id>",
      "name": "<resolved_entity_name>"
    }
  ]
}

Demonstration Example (For schema reference ONLY):
Input History: user discusses "Laptop Pro 15", then asks "how much is it?"
Output:
{
  "references": {
    "it": "Laptop Pro 15"
  },
  "entities": [
    {
      "type": "product",
      "id": "p_1",
      "name": "Laptop Pro 15"
    }
  ]
}`

	var histBuilder strings.Builder
	histBuilder.WriteString("Current Room State: " + stateJSON + "\n\nRecent History:\n")
	for _, m := range history {
		histBuilder.WriteString(fmt.Sprintf("%s: %s\n", m.Role, m.Content))
	}

	reqBody := chatCompletionRequest{
		Model: s.cfg.OpenRouterReasoningModel,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: histBuilder.String()},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.cfg.OpenRouterAPIKey))

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, nil, fmt.Errorf("openrouter resolve references returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var respBody chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, nil, err
	}

	if len(respBody.Choices) == 0 {
		return nil, nil, fmt.Errorf("openrouter resolve references returned empty choices")
	}

	content := respBody.Choices[0].Message.Content
	// Clean markdown wrappers if present
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var refRes referenceJSON
	if err := json.Unmarshal([]byte(content), &refRes); err != nil {
		// Non-fatal: log warning and return empty map
		return map[string]string{}, []domain.Entity{}, nil
	}

	if refRes.References == nil {
		refRes.References = make(map[string]string)
	}

	return refRes.References, refRes.Entities, nil
}

func (s *openRouterServiceClient) EvaluateContext(ctx context.Context, query string, contextPkg *domain.ContextPackage) (*domain.ContextEvaluation, error) {
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(s.cfg.OpenRouterBaseURL, "/"))

	systemPrompt := `You are a Context Quality Evaluator for an AI agent infrastructure system.
Your job is to evaluate the provided "Context Package" against the user's latest "Query".

Assess if the user's query represents a "New Topic Shift" (i.e. the user is asking about something completely unrelated to the past history and state variables stored in the Context Package, such as asking about Hanoi hotels when the past context was about Tokyo). If it is a new topic:
- set "is_new_topic" to true.
- "relevance_score" and "completeness_score" should be 1.0 (since no prior context exists to retrieve, the system did not fail).

Otherwise:
- set "is_new_topic" to false.
- evaluate relevance and completeness normally.

Definitions for the dimensions (scale from 0.0 to 1.0):
1. relevance_score: Does the context contain the appropriate historical background, state, or entities needed to interpret/answer the query? If it's a new topic, default to 1.0.
2. completeness_score: Does the context contain ALL necessary historical references, active topics, or state variables previously discussed that are needed for this query? If it's a new topic, default to 1.0.
3. token_efficiency_score: Is the context concise and focused, or does it contain a lot of bloated/irrelevant information? (1.0 = highly efficient, no fluff. 0.0 = completely bloated with unrelated history/state).

Output ONLY a valid JSON object matching this schema:
{
  "relevance_score": 0.9,
  "completeness_score": 0.8,
  "token_efficiency_score": 0.95,
  "reasoning": "Brief explanation of the scores.",
  "is_new_topic": false
}`

	// Prepare a cleaned context package for the LLM and snapshot (strip embeddings to save token costs, but retain memory content for accurate evaluation)
	evalPkg := *contextPkg
	if len(evalPkg.Memories) > 0 {
		cleanedMem := make([]domain.Memory, len(evalPkg.Memories))
		for i, m := range evalPkg.Memories {
			c := m
			c.Embedding = nil
			cleanedMem[i] = c
		}
		evalPkg.Memories = cleanedMem
	}
	contextJSON, _ := json.Marshal(evalPkg)

	userPrompt := fmt.Sprintf("User Query: %s\n\nContext Package:\n%s", query, string(contextJSON))

	reqBody := chatCompletionRequest{
		Model: s.cfg.OpenRouterEvaluationModel,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
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
	req.Header.Set("Authorization", "Bearer "+s.cfg.OpenRouterAPIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openrouter evaluate context api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var respBody chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return nil, err
	}

	if len(respBody.Choices) == 0 {
		return nil, fmt.Errorf("openrouter evaluate context api returned empty choices")
	}

	content := respBody.Choices[0].Message.Content

	var eval domain.ContextEvaluation
	if err := json.Unmarshal([]byte(content), &eval); err != nil {
		return nil, fmt.Errorf("failed to parse JSON evaluation response: %w", err)
	}

	eval.Query = query
	eval.ContextSnapshot = contextJSON

	return &eval, nil
}
