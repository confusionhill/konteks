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
	Summary         string          `json:"summary"`
	ActiveTopic     string          `json:"active_topic"`
	State           json.RawMessage `json:"state"`
	NewSalientFacts []string        `json:"new_salient_facts"`
}

func (s *openRouterServiceClient) ExtractStateAndSummary(ctx context.Context, history []domain.Message, currentSummary string, currentStateJSON string, schemaJSON string) (string, string, string, []string, error) {
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(s.cfg.OpenRouterBaseURL, "/"))

	systemPrompt := `You are a conversation state and summary extraction engine. Analyze the conversation history, current summary, and current state to produce an updated state package.
The output must be a valid JSON object matching exactly this schema:
{
  "summary": "a concise description of the conversation progress",
  "active_topic": "the primary active topic (e.g. hotel search, flight booking, payment)",
  "state": {},
  "new_salient_facts": ["fact 1", "fact 2"]
}

Guidelines:
1. 'summary': Keep it concise but ensure it captures context (e.g., "User is comparing budget hotels in Bali").
2. 'active_topic': Briefly name the active user intent.
3. 'state': A flat JSON object containing key-value pairs representing parameters discussed so far. Retain existing state parameters unless explicitly updated or refuted. CRITICAL: DROP or REMOVE state parameters that have been fulfilled, abandoned, completed, or are no longer relevant to the active topic to conserve context space.
4. 'new_salient_facts': Extract any NEW, distinct facts, preferences, or entities stated by the user IN THIS MOST RECENT TURN that should be remembered long-term. Output them as concise, declarative sentences (e.g. 'User is allergic to peanuts'). Do not repeat facts already in the summary. When in doubt, extract the fact. Err on the side of over-extraction, paying special attention to constraints, dates, names, locations, and user preferences. If there is absolutely nothing worth remembering (e.g., greetings, filler), output an empty array.`

	if schemaJSON != "" && schemaJSON != "{}" && schemaJSON != "null" {
		systemPrompt += fmt.Sprintf("\n\nCRITICAL: The 'state' object in the output MUST adhere strictly to this schema: %s", schemaJSON)
	}

	// Formulate history string
	var histBuilder strings.Builder
	histBuilder.WriteString("Current Summary: " + currentSummary + "\n")
	if currentStateJSON != "" && currentStateJSON != "{}" && currentStateJSON != "null" {
		histBuilder.WriteString("Current State: " + currentStateJSON + "\n")
	}
	histBuilder.WriteString("\nRecent Conversation History:\n")
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
		return "", "", "", nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", "", "", nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", s.cfg.OpenRouterAPIKey))

	resp, err := s.client.Do(req)
	if err != nil {
		return "", "", "", nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", "", "", nil, fmt.Errorf("openrouter chat completion api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var respBody chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return "", "", "", nil, err
	}

	if len(respBody.Choices) == 0 {
		return "", "", "", nil, fmt.Errorf("openrouter chat completion api returned empty choices")
	}

	content := respBody.Choices[0].Message.Content

	var extraction extractionJSON
	if err := json.Unmarshal([]byte(content), &extraction); err != nil {
		return "", "", "", nil, fmt.Errorf("failed to parse JSON extraction response: %w. raw content: %s", err, content)
	}

	stateStr := string(extraction.State)
	if stateStr == "" || stateStr == "null" {
		stateStr = "{}"
	}

	if extraction.NewSalientFacts == nil {
		extraction.NewSalientFacts = []string{}
	}

	return extraction.Summary, extraction.ActiveTopic, stateStr, extraction.NewSalientFacts, nil
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

type contradictionJSON struct {
	IsContradiction bool    `json:"is_contradiction"`
	Confidence      float64 `json:"confidence"`
	Reasoning       string  `json:"reasoning"`
}

func (s *openRouterServiceClient) EvaluateContradiction(ctx context.Context, factA string, factB string) (bool, float64, error) {
	url := fmt.Sprintf("%s/chat/completions", strings.TrimSuffix(s.cfg.OpenRouterBaseURL, "/"))

	systemPrompt := `You are an Entity-Attribute-Value (EAV) Fact Contradiction Evaluator.
Compare Fact A (older stored fact) and Fact B (newer user fact) to determine if Fact B explicitly SUPERSEDES or CONTRADICTS Fact A on the exact same entity attribute.

Rules:
1. Contradiction occurs when Fact B changes or overrides an attribute of Fact A (e.g. "User lives in SF" vs "User lives in Tokyo", or "DB is Postgres" vs "DB switched to MySQL").
2. Coexistence (NOT a contradiction) occurs when Fact B adds complementary info without overriding Fact A (e.g. "User lives in SF" vs "User works remotely", or "Prefers Go" vs "Uses Python for scripts").
3. Set confidence between 0.0 and 1.0. High confidence (>=0.90) is required for explicit supersedence.

Output ONLY a valid JSON object matching this schema:
{
  "is_contradiction": true,
  "confidence": 0.95,
  "reasoning": "Fact B updates the user's primary residence from Tokyo to SF."
}`

	userPrompt := fmt.Sprintf("Fact A (Older): %s\nFact B (Newer): %s", factA, factB)

	reqBody := chatCompletionRequest{
		Model: s.cfg.OpenRouterReasoningModel,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		ResponseFormat: &responseFormat{Type: "json_object"},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return false, 0.0, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return false, 0.0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.cfg.OpenRouterAPIKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return false, 0.0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return false, 0.0, fmt.Errorf("openrouter evaluate contradiction api returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var respBody chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		return false, 0.0, err
	}

	if len(respBody.Choices) == 0 {
		return false, 0.0, fmt.Errorf("openrouter evaluate contradiction api returned empty choices")
	}

	content := respBody.Choices[0].Message.Content

	var res contradictionJSON
	if err := json.Unmarshal([]byte(content), &res); err != nil {
		return false, 0.0, fmt.Errorf("failed to parse JSON contradiction response: %w", err)
	}

	return res.IsContradiction, res.Confidence, nil
}
