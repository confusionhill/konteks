package worker

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/dika/contextkit/backend/internal/domain"
)

type stubMemoryUpdater struct {
	mu   sync.Mutex
	ids  []string
	err  error
}

func (s *stubMemoryUpdater) UpdateEmbedding(ctx context.Context, memoryID string, embedding []float32) error {
	if s.err != nil {
		return s.err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ids = append(s.ids, memoryID)
	return nil
}

type stubOpenRouterClient struct {
	embedding []float32
	err       error
}

func (c *stubOpenRouterClient) GenerateEmbedding(ctx context.Context, text string) ([]float32, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.embedding, nil
}

func (c *stubOpenRouterClient) ExtractStateAndSummary(ctx context.Context, history []domain.Message, currentSummary string, currentStateJSON string, schemaJSON string) (string, string, string, []string, error) {
	return "", "", "{}", nil, nil
}

func (c *stubOpenRouterClient) EvaluateContext(ctx context.Context, query string, contextPkg *domain.ContextPackage) (*domain.ContextEvaluation, error) {
	return nil, nil
}

func (c *stubOpenRouterClient) EvaluateContradiction(ctx context.Context, factA string, factB string) (bool, float64, error) {
	return false, 0, nil
}

func TestMCPEmbeddingWorker_GeneratesAndPersistsEmbedding(t *testing.T) {
	updater := &stubMemoryUpdater{}
	client := &stubOpenRouterClient{embedding: []float32{0.1, 0.2}}
	w := NewMCPEmbeddingWorker(client, updater, 4)
	w.Start(2)
	defer w.Close()

	if err := w.Enqueue(MCPMemoryEmbedTask{ProjectID: "proj-a", MemoryID: "mem-1", Content: "fact"}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}

	// Close so the worker drains the queue before we assert.
	w.Close()
	if len(updater.ids) != 1 || updater.ids[0] != "mem-1" {
		t.Fatalf("expected memory mem-1 to be updated, got %+v", updater.ids)
	}
}

func TestMCPEmbeddingWorker_EnqueueAfterClose_Errors(t *testing.T) {
	w := NewMCPEmbeddingWorker(&stubOpenRouterClient{}, &stubMemoryUpdater{}, 4)
	w.Start(1)
	w.Close()

	err := w.Enqueue(MCPMemoryEmbedTask{MemoryID: "mem-1"})
	if !errors.Is(err, errWorkerClosed) {
		t.Fatalf("expected errWorkerClosed, got %v", err)
	}
}

func TestMCPEmbeddingWorker_EmbeddingError_IsSkipped(t *testing.T) {
	updater := &stubMemoryUpdater{}
	client := &stubOpenRouterClient{err: errors.New("embedding failed")}
	w := NewMCPEmbeddingWorker(client, updater, 4)
	w.Start(1)

	if err := w.Enqueue(MCPMemoryEmbedTask{MemoryID: "mem-1", Content: "fact"}); err != nil {
		t.Fatalf("enqueue failed: %v", err)
	}
	w.Close()
	if len(updater.ids) != 0 {
		t.Fatalf("memory should not be updated on embedding failure, got %+v", updater.ids)
	}
}