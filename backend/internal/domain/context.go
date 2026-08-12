package domain

import (
	"context"
	"encoding/json"
)

type Entity struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type ContextPolicy struct {
	MaxTokens         int     `json:"max_tokens,omitempty"`
	MinRelevanceScore float64 `json:"min_relevance_score,omitempty"`
	Strategy          string  `json:"strategy,omitempty"` // "auto" (default), "recent_heavy", "memory_heavy", "hybrid"
	MaxHistoryDepth   int     `json:"max_history_depth,omitempty"`
	FreshnessWeight   float64 `json:"freshness_weight,omitempty"`
}

type GetContextRequest struct {
	Query         string          `json:"query"`
	Policy        ContextPolicy   `json:"policy"`
	DynamicSchema json.RawMessage `json:"dynamic_schema,omitempty"`
}

type ContextPackage struct {
	RoomID           string            `json:"room_id"`
	Summary          string            `json:"summary"`
	ActiveTopic      string            `json:"active_topic"`
	State            any               `json:"state"`
	RelevantMessages []Message         `json:"relevant_messages"`
	Memories         []Memory          `json:"memories"`
	Entities         []Entity          `json:"entities"`
	References       map[string]string `json:"references,omitempty"`
	StrategyUsed     string            `json:"strategy_used,omitempty"`
	TokensEstimated  int               `json:"tokens_estimated,omitempty"`
	TokensProcessed  int               `json:"tokens_processed,omitempty"`
}

type OpenRouterServiceClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	ExtractStateAndSummary(ctx context.Context, history []Message, currentSummary string, schemaJSON string) (newSummary string, activeTopic string, updatedState string, err error)
	ResolveReferences(ctx context.Context, history []Message, stateJSON string) (references map[string]string, entities []Entity, err error)
}

type ContextUsecase interface {
	GetContextPackage(ctx context.Context, tenantID string, roomID string, req GetContextRequest) (*ContextPackage, error)
}
