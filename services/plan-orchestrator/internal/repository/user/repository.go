package user

import (
	"context"
	"errors"
	"sync"

	"github.com/travel-agent/services/plan-orchestrator/internal/domain"
)

var ErrUserNotFound = errors.New("user not found")

type Repository interface {
	Create(context.Context, domain.User) (domain.User, error)
	GetByID(context.Context, int64) (domain.User, error)
	GetByUsername(context.Context, string) (domain.User, error)
}

type InMemoryRepository struct {
	mu         sync.RWMutex
	users      map[int64]domain.User
	byUsername map[string]int64
	nextID     int64
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		users:      make(map[int64]domain.User),
		byUsername: make(map[string]int64),
		nextID:     1,
	}
}

func (r *InMemoryRepository) Create(_ context.Context, user domain.User) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	user.ID = r.nextID
	r.nextID++
	r.users[user.ID] = user
	r.byUsername[user.Username] = user.ID
	return user, nil
}

func (r *InMemoryRepository) GetByID(_ context.Context, id int64) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[id]
	if !ok {
		return domain.User{}, ErrUserNotFound
	}
	return user, nil
}

func (r *InMemoryRepository) GetByUsername(_ context.Context, username string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	id, ok := r.byUsername[username]
	if !ok {
		return domain.User{}, ErrUserNotFound
	}
	return r.users[id], nil
}
