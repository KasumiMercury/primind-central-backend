package repository

import (
	"context"
	"sync"
	"time"

	"github.com/KasumiMercury/primind-central-backend/internal/auth/domain/user"
)

type UserRecord struct {
	user      *user.User
	createdAt time.Time
}

type inMemoryUserRepository struct {
	mu   sync.Mutex
	byID map[user.ID]UserRecord
}

func NewInMemoryUserRepository() user.UserRepository {
	return &inMemoryUserRepository{
		byID: make(map[user.ID]UserRecord),
	}
}

func (r *inMemoryUserRepository) SaveUser(_ context.Context, u *user.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.byID[u.ID()] = UserRecord{
		user:      u,
		createdAt: time.Now(),
	}
	return nil
}

func (r *inMemoryUserRepository) GetUserByID(_ context.Context, id user.ID) (*user.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, exists := r.byID[id]
	if !exists {
		return nil, user.ErrUserNotFound
	}
	return record.user, nil
}
