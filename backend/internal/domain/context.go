package domain

import (
	"context"
	"encoding/json"
	"time"
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
	FastMode          bool    `json:"fast_mode,omitempty"`
}

type GetContextRequest struct {
	Query              string          `json:"query"`
	Policy             ContextPolicy   `json:"policy"`
	DynamicSchema      json.RawMessage `json:"dynamic_schema,omitempty"`
	IsEncrypted        bool            `json:"is_encrypted"`
	EncryptionMetadata *string         `json:"encryption_metadata,omitempty"`
	IncludeChildren    bool            `json:"include_children,omitempty"`
	RequestKey         string          `json:"request_key,omitempty"`
}

type ContextPackage struct {
	RoomID           string               `json:"room_id"`
	Summary          string               `json:"summary"`
	ActiveTopic      string               `json:"active_topic"`
	State            any                  `json:"state"`
	RelevantMessages []Message            `json:"relevant_messages"`
	Memories         []Memory             `json:"memories,omitempty"`
	Entities         []Entity             `json:"entities"`
	References       map[string]string    `json:"references,omitempty"`
	ChildAgents      []ChildAgentOverview `json:"child_agents,omitempty"`
	StrategyUsed     string               `json:"strategy_used,omitempty"`
	TokensEstimated  int                  `json:"tokens_estimated,omitempty"`
	TokensProcessed  int                  `json:"tokens_processed,omitempty"`
}

type OpenRouterServiceClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	ExtractStateAndSummary(ctx context.Context, history []Message, currentSummary string, schemaJSON string) (newSummary string, activeTopic string, updatedState string, salientFacts []string, err error)
	EvaluateContext(ctx context.Context, query string, contextPkg *ContextPackage) (*ContextEvaluation, error)
}

type ContextEvaluation struct {
	ID                   string          `json:"id"`
	ProjectID            string          `json:"project_id"`
	RoomID               string          `json:"room_id"`
	Query                string          `json:"query"`
	ContextSnapshot      json.RawMessage `json:"context_snapshot"`
	RelevanceScore       float64         `json:"relevance_score"`
	CompletenessScore    float64         `json:"completeness_score"`
	TokenEfficiencyScore float64         `json:"token_efficiency_score"`
	Reasoning            string          `json:"reasoning"`
	IsNewTopic           bool            `json:"is_new_topic"`
	IsEncrypted          bool            `json:"is_encrypted"`
	EncryptionMetadata   *string         `json:"encryption_metadata,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
}

type EvaluationSummary struct {
	AvgRelevance       float64 `json:"avg_relevance"`
	AvgCompleteness    float64 `json:"avg_completeness"`
	AvgTokenEfficiency float64 `json:"avg_token_efficiency"`
	TotalEvaluations   int     `json:"total_evaluations"`
}

type EvaluationRepository interface {
	Create(ctx context.Context, eval *ContextEvaluation) error
	GetByID(ctx context.Context, evalID string) (*ContextEvaluation, error)
	GetByRoomID(ctx context.Context, roomID string) ([]ContextEvaluation, error)
	GetByProjectID(ctx context.Context, projectID string, limit int, offset int) ([]ContextEvaluation, int, error)
	GetSummary(ctx context.Context, projectID string) (*EvaluationSummary, error)
}

type ContextUsecase interface {
	GetContextPackage(ctx context.Context, projectID string, roomID string, req GetContextRequest) (*ContextPackage, error)
	Warmup(ctx context.Context, projectID string, roomID string) error
}
