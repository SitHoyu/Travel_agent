package plan

import (
	"context"
	"sync"

	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
)

type Repository interface {
	Create(context.Context, domain.PlanRecord) (domain.PlanRecord, error)
	ListByUserID(context.Context, int64, int, int) ([]domain.PlanRecord, int64, error)
	GetByIDAndUserID(context.Context, int64, int64) (domain.PlanRecord, bool, error)
}

type InMemoryRepository struct {
	mu      sync.RWMutex
	records map[int64]domain.PlanRecord
	nextID  int64
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		records: make(map[int64]domain.PlanRecord),
		nextID:  1,
	}
}

func (r *InMemoryRepository) Create(_ context.Context, record domain.PlanRecord) (domain.PlanRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record.ID = r.nextID
	r.nextID++
	r.records[record.ID] = record
	return record, nil
}

func (r *InMemoryRepository) ListByUserID(_ context.Context, userID int64, page, pageSize int) ([]domain.PlanRecord, int64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	all := make([]domain.PlanRecord, 0)
	for _, record := range r.records {
		if record.UserID == userID {
			all = append(all, record)
		}
	}

	total := int64(len(all))
	start := (page - 1) * pageSize
	if start >= len(all) {
		return []domain.PlanRecord{}, total, nil
	}

	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}

	return all[start:end], total, nil
}

func (r *InMemoryRepository) GetByIDAndUserID(_ context.Context, id, userID int64) (domain.PlanRecord, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	record, ok := r.records[id]
	if !ok || record.UserID != userID {
		return domain.PlanRecord{}, false, nil
	}
	return record, true, nil
}
