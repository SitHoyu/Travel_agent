package plan

import (
	"context"
	"sync"

	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
)

type Repository interface {
	Save(context.Context, domain.PlanRunResult) error
	Get(context.Context, string) (domain.PlanRunResult, bool)
}

type InMemoryRepository struct {
	mu      sync.RWMutex
	records map[string]domain.PlanRunResult
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		records: make(map[string]domain.PlanRunResult),
	}
}

func (r *InMemoryRepository) Save(_ context.Context, result domain.PlanRunResult) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.records[result.SessionID] = result
	return nil
}

func (r *InMemoryRepository) Get(_ context.Context, sessionID string) (domain.PlanRunResult, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result, ok := r.records[sessionID]
	return result, ok
}
