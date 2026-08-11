package domain

import "context"

type Entity struct {
	Type string `json:"type"`
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
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
}

type OpenRouterServiceClient interface {
	GenerateEmbedding(ctx context.Context, text string) ([]float32, error)
	ExtractStateAndSummary(ctx context.Context, history []Message, currentSummary string) (newSummary string, activeTopic string, updatedState string, err error)
}

type ContextUsecase interface {
	GetContextPackage(ctx context.Context, tenantID string, roomID string, query string, maxTokens int) (*ContextPackage, error)
}
